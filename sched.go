// Copyright 2018 Changkun Ou. All rights reserved.
// Use of this source code is governed by a MIT
// license that can be found in the LICENSE file.

package sched

import (
	"encoding/json"
	"reflect"
	"sync"
	"time"

	"github.com/changkun/sched/internal/pool"
	"github.com/changkun/sched/internal/pq"
	"github.com/changkun/sched/internal/store"
	"github.com/changkun/sched/task"
)

var (
	worker        *Scheduler
	schedulerOnce sync.Once
)

// Scheduler is the actual scheduler for task scheduling
type Scheduler struct {
	mu    *sync.Mutex
	timer *time.Timer
	queue *pq.TaskQueue
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
			queue: pq.NewTaskQueue(),
		}
	})
	return worker
}

// Recover given tasks
func Recover(ts ...task.Interface) error {
	return New().Recover(ts...)
}

// Recover given tasks
func (s *Scheduler) Recover(ts ...task.Interface) error {
	for i := range ts {
		if err := s.recover(ts[i]); err != nil {
			return err
		}
	}
	return nil
}

func (s *Scheduler) recover(t task.Interface) error {
	ids, err := store.GetRecords()
	if err != nil {
		return err
	}

	for i := range ids {
		r := &store.Record{}
		if err := r.Read(ids[i]); err != nil {
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
		temp2.SetID(ids[i])
		temp2.SetExecution(r.Execution)
		s.queue.Push(temp2)
	}
	s.resume()
	return nil
}

// Setup given tasks
func Setup(ts ...task.Interface) error {
	return New().Setup(ts...)
}

// Setup given tasks
func (s *Scheduler) Setup(ts ...task.Interface) error {
	for i := range ts {
		if err := s.setup(ts[i]); err != nil {
			return err
		}
	}
	return nil
}

func (s *Scheduler) setup(t task.Interface) error {
	// save asap
	if err := store.Save(t); err != nil {
		return err
	}
	s.schedule(t)
	return nil
}

// Launch given tasks immediately
func Launch(ts ...task.Interface) error {
	return New().Launch(ts...)
}

// Launch given tasks immediately
func (s *Scheduler) Launch(ts ...task.Interface) error {
	for i := range ts {
		if err := s.launch(ts[i]); err != nil {
			return err
		}
	}
	return nil
}

func (s *Scheduler) launch(t task.Interface) error {
	t.SetExecution(time.Now().UTC())
	return s.Setup(t)
}

func (s *Scheduler) schedule(t task.Interface) {
	s.pause()
	defer s.resume()

	// if priority is able to be update
	if s.queue.Update(t) {
		return
	}

	s.queue.Push(t)
}

func (s *Scheduler) setTimer(duration time.Duration) {
	s.mu.Lock()
	s.timer = time.NewTimer(duration)
	s.mu.Unlock()
}

func (s *Scheduler) getTimer() *time.Timer {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.timer
}

func (s *Scheduler) pause() {
	s.mu.Lock()
	if s.timer != nil {
		s.timer.Stop()
	}
	s.mu.Unlock()
}

func (s *Scheduler) resume() {
	t := s.queue.Peek()
	if t == nil {
		return
	}
	s.setTimer(t.GetExecution().Sub(time.Now().UTC()))
	go func() {
		<-s.getTimer().C
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

	s.execute(t)

	if err := store.Unlock(t); err != nil {
		// no need to hadle unlock fail
		// since there is a timeout on redis
		return
	}
}

func (s *Scheduler) verify(t task.Interface) (*time.Time, error) {
	r := &store.Record{}
	if err := r.Read(t.GetID()); err != nil {
		return nil, err
	}
	taskTime := t.GetExecution()
	if taskTime.Before(r.Execution) {
		return &r.Execution, nil
	}
	return &taskTime, nil
}

func (s *Scheduler) retry(t task.Interface) {
	t.SetExecution(t.GetExecution().Add(t.GetRetryDuration()))
	if err := s.Setup(t); err != nil {
		// If Setup() is fail because of redis failure,
		// directly schedule the task without any hesitate
		// In case it may leads a problem of unabiding task
		// scheduling
		s.schedule(t)
	}
}

func (s *Scheduler) execute(t task.Interface) {
	execution, err := s.verify(t)
	if err != nil {
		return
	}
	// for timer tollerance
	if execution.Sub(time.Now().UTC()) < time.Millisecond {
		retry, err := t.Execute()
		if retry || err != nil {
			s.retry(t)
			return
		}
		if err := store.Delete(t); err != nil {
			// NOTE: Generally this is not able to be fail.
			// However it may caused by the lost of connection.
			// Though task will be recovered then app restarts,
			// but it may leads an inconsistency, we need other
			// means to solve this problem
			return
		}
		return
	}
	// reschedule task, we must save the task again by using s.Setup
	t.SetExecution(*execution)
	s.Setup(t)
}
