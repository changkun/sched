package goscheduler

import (
	"encoding/json"
	"reflect"
	"sync"
	"time"

	"github.com/changkun/goscheduler/internal/pool"
	"github.com/changkun/goscheduler/internal/pq"
	"github.com/changkun/goscheduler/internal/store"
	"github.com/changkun/goscheduler/task"
)

var (
	worker        *Scheduler
	schedulerOnce sync.Once
)

// Scheduler is the actual scheduler for task scheduling
type Scheduler struct {
	mu    *sync.Mutex
	timer *time.Timer
	queue *pq.TimerTaskQueue
}

// Init internal connection pool to data store
func Init(url string) {
	pool.Init(url)
}

// New creates a temporary task scheduler
func New() *Scheduler {
	schedulerOnce.Do(func() {
		worker = &Scheduler{
			mu:    &sync.Mutex{},
			timer: nil,
			queue: pq.NewTimerTaskQueue(),
		}
	})
	return worker
}

// RecoverAll given tasks
func (s *Scheduler) RecoverAll(ts ...task.Interface) error {
	for _, t := range ts {
		if err := s.Recover(t); err != nil {
			return err
		}
	}
	return nil
}

// Recover given task type
func (s *Scheduler) Recover(t task.Interface) error {
	ids, err := store.GetRecords()
	if err != nil {
		return err
	}

	for _, id := range ids {
		r := &store.Record{}
		if err := r.Read(id); err != nil {
			return err
		}

		data, _ := json.Marshal(r.Data)

		// Note: The following comparasion provides a generic mechanism in golang, which
		// unmarshals an unknown type of data into acorss multiple into an arbitrary variable.
		//
		// temp1 holds for a unset value of t, and temp2 tries to be set by json.Unmarshal.
		//
		// In the end, if temp1 and temp2 are appropriate type of tasks, then temp2 should
		// not DeepEqual to temp1, because temp2 is setted by store data.
		// Otherwise, the determined task is inappropriate type to be scheduled, jump to
		// next record and see if it can be scheduled.
		temp1 := reflect.New(reflect.ValueOf(t).Elem().Type()).Interface().(task.Interface)
		temp2 := reflect.New(reflect.ValueOf(t).Elem().Type()).Interface().(task.Interface)
		if err := json.Unmarshal(data, &temp2); err != nil {
			continue
		}
		if reflect.DeepEqual(temp1, temp2) {
			continue
		}
		temp2.SetID(id)
		temp2.SetExecution(r.Execution)
		s.queue.Push(temp2)
	}
	s.resume()
	return nil
}

// SetupAll given tasks
func (s *Scheduler) SetupAll(ts ...task.Interface) error {
	for _, t := range ts {
		if err := s.Setup(t); err != nil {
			return err
		}
	}
	return nil
}

// Setup a task into runner queue
func (s *Scheduler) Setup(t task.Interface) error {
	// save asap
	store.Save(t)

	s.pause()
	defer s.resume()

	// if priority is able to be update
	if s.queue.Update(t) {
		return nil
	}

	s.queue.Push(t)
	return nil
}

// LaunchAll given tasks
func (s *Scheduler) LaunchAll(ts ...task.Interface) error {
	for _, t := range ts {
		if err := s.Launch(t); err != nil {
			return err
		}
	}
	return nil
}

// Launch a task immediately
func (s *Scheduler) Launch(t task.Interface) error {
	t.SetExecution(time.Now().UTC())
	return s.Setup(t)
}

func (s *Scheduler) pause() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.timer != nil {
		s.timer.Stop()
	}
}

func (s *Scheduler) resume() {
	t := s.queue.Peek()
	if t == nil {
		return
	}
	s.mu.Lock()
	s.timer = time.NewTimer(t.GetExecution().Sub(time.Now().UTC()))
	timer := s.timer
	s.mu.Unlock()
	go func() {
		<-timer.C
		t := s.queue.Pop()
		if t == nil {
			return
		}
		s.resume()
		s.arrival(t)
	}()
}

func (s *Scheduler) arrival(t task.Interface) {
	ok, err := store.Lock(t)
	if err != nil || !ok {
		return
	}
	defer store.Unlock(t)

	s.execute(t)
}

func (s *Scheduler) verify(t task.Interface) (*time.Time, error) {
	r := &store.Record{}
	r.Read(t.GetID())
	taskTime := t.GetExecution()
	if taskTime.Before(r.Execution) {
		return &r.Execution, nil
	}
	return &taskTime, nil
}
func (s *Scheduler) execute(t task.Interface) {
	execution, err := s.verify(t)
	if err != nil {
		return
	}
	if execution.Before(time.Now().UTC()) {
		retry, err := t.Execute()
		if retry || err != nil {
			s.retry(t)
			return
		}
		// TODO: this may fail, how to solve
		store.Delete(t)
		return
	}
	s.mu.Lock()
	s.queue.Push(t)
	s.mu.Unlock()
}

func (s *Scheduler) retry(t task.Interface) {
	t.SetExecution(t.GetExecution().Add(t.GetRetryDuration()))
	s.Setup(t)
}
