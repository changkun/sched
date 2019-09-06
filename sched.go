// Copyright 2018 Changkun Ou. All rights reserved.
// Use of this source code is governed by a MIT
// license that can be found in the LICENSE file.

package sched

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"sync/atomic"
	"time"
	"unsafe"
)

// Init initialize task scheduler
func Init(db string, all ...Task) ([]TaskFuture, error) {
	c, err := newCache(db)
	if err != nil {
		return nil, err
	}
	sched0 = &sched{
		timer: unsafe.Pointer(time.NewTimer(0)),
		tasks: newTaskQueue(),
		cache: c,
	}
	return sched0.recover(all...)
}

// Stop stops runtime scheduler gracefully.
// Note that the call should only be called then application terminates
func Stop() {
	// pause sched0 fisrt.
	Pause()

	// wait until all started tasks (i.e. tasks is executing other than
	// timing) stops
	for atomic.LoadUint64(&sched0.running) > 0 {
	}

	// reset pausing indicator
	atomic.AddUint64(&sched0.pausing, ^uint64(0))

	sched0.cache.Close()
}

// Wait waits all tasks to be scheduled.
func Wait() {
	// With function call, no need for runtime.Gosched()
	for sched0.tasks.length() != 0 {
	}
}

// Submit given tasks
func Submit(t Task) (TaskFuture, error) {
	return sched0.submit(t)
}

// Trigger given tasks immediately
func Trigger(t Task) (TaskFuture, error) {
	return sched0.trigger(t)
}

// Pause stops the sched from running,
// this is a pair call with Resume(), Pause() must be called first
//
// Pause() is the only way that completely pause sched from running.
// the internal sched0.pause() is only used for internal scheduling,
// which is not a real pause.
func Pause() {
	atomic.AddUint64(&sched0.pausing, 1)
	sched0.pause()
}

// Resume resumes sched and start executing tasks
// this is a pair call with Pause(), Resume() must be called second
func Resume() {
	atomic.AddUint64(&sched0.pausing, ^uint64(0)) // -1
	sched0.resume()
}

var sched0 *sched

// sched is the actual scheduler for task scheduling
//
// sched implements greedy scheduling, with a timer and a task queue,
// the task queue is a priority queue that orders tasks by its executing time.
// the timer is the only time.Timer lives in runtime, it serves the head
// task in the task queue.
type sched struct {
	// running counts the tasks already starts that cannot be stopped,
	// for timing tasks that still waiting for execution, call sched.tasks.Len().
	running uint64 // atomic
	// pausing is a sign that indicates if sched should stop running.
	pausing uint64 // atomic
	// timer is the only timer during the runtime
	timer unsafe.Pointer // *time.Timer
	// cancel cancels a timer if a timer need to reset
	cancel atomic.Value // context.CancelFunc
	// tasks is a TaskQueue that stores all unscheduled tasks in memory
	tasks *taskQueue
	// cache store
	cache *cache
}

func (s *sched) recover(ts ...Task) (futures []TaskFuture, err error) {
	ids, err := getRecords()
	if err != nil {
		return nil, err
	}
	for _, t := range ts {
		for i := range ids {
			future, err := s.load(ids[i], t)
			if future != nil && err == nil {
				futures = append(futures, future)
			}
		}
	}
	s.resume()
	return
}

func (s *sched) load(id string, t Task) (TaskFuture, error) {
	r := &record{ID: id}
	if err := r.read(); err != nil {
		return nil, err
	}

	data, _ := json.Marshal(r.Data)

	// Note: The following comparison provides a generic mechanism in golang, which
	// unmarshals an unknown type of data into acorss multiple into an arbitrary variable.
	//
	// temp1 holds for a unset value of t, and temp2 tries to be set by json.Unmarshal.
	//
	// In the end, if temp1 and temp2 are appropriate type of tasks, then temp2 should
	// not DeepEqual to temp1, because temp2 is setted by store data.
	// Otherwise, the determined task is inappropriate type to be scheduled, jump to
	// next record and see if it can be scheduled.
	temp1 := reflect.New(reflect.ValueOf(t).Elem().Type()).Interface().(Task)
	temp2 := reflect.New(reflect.ValueOf(t).Elem().Type()).Interface().(Task)
	json.Unmarshal(data, &temp2)
	if reflect.DeepEqual(temp1, temp2) || temp2 == nil || !temp2.IsValidID() {
		return nil, nil
	}
	temp2.SetID(id)
	temp2.SetExecution(r.Execution)
	future, _ := s.tasks.push(temp2)
	return future, nil
}

// submit given tasks
func (s *sched) submit(t Task) (future TaskFuture, err error) {
	// save asap
	if err = save(t); err != nil {
		return
	}
	future = s.schedule(t)
	return
}

// trigger given tasks immediately
func (s *sched) trigger(t Task) (TaskFuture, error) {
	t.SetExecution(time.Now().UTC())
	return s.submit(t)
}

func (s *sched) schedule(t Task) TaskFuture {
	s.pause()
	defer s.resume()

	// if priority is able to be update
	if future, ok := s.tasks.update(t); ok {
		return future
	}

	future, _ := s.tasks.push(t)
	return future
}

func (s *sched) setTimer(d time.Duration) {
	// spin lock
	for {
		// fast path: reuse the timer
		old := atomic.LoadPointer(&s.timer)
		if (*time.Timer)(old).Stop() {
			(*time.Timer)(old).Reset(d)
			return
		}

		if atomic.CompareAndSwapPointer(&s.timer, old, unsafe.Pointer(time.NewTimer(d))) {
			(*time.Timer)(old).Stop()
			return
		}
	}
}

func (s *sched) getTimer() *time.Timer {
	return (*time.Timer)(atomic.LoadPointer(&s.timer))
}

// pause pauses sched timer, it does not concurrently pause tasks from running.
// Thus, do NOT call this for complete pausing sched, call Pause() instead.
func (s *sched) pause() {
	(*time.Timer)(atomic.LoadPointer(&s.timer)).Stop()
}

func (s *sched) ispausing() bool {
	return atomic.LoadUint64(&s.pausing) > 0
}

func (s *sched) resume() {
	t := s.tasks.peek()
	if t == nil {
		if x, ok := s.cancel.Load().(context.CancelFunc); ok {
			x()
		}
		return
	}
	s.setTimer(t.GetExecution().Sub(time.Now().UTC()))
	ctx, cancel := context.WithCancel(context.Background())
	if x, ok := s.cancel.Load().(context.CancelFunc); ok {
		x()
	}
	s.cancel.Store(cancel)
	go s.timing(ctx)
}

func (s *sched) timing(ctx context.Context) {
	timer := s.getTimer()
	if timer == nil {
		return
	}
	select {
	case <-timer.C:
		s.worker()
	case <-ctx.Done():
	}
}
func (s *sched) worker() {
	// fast path.
	// if sched requires pausing, then stop executing and resume it.
	if s.ispausing() {
		return
	}

	// medium path.
	// stop execution if task queue is empty
	task := s.tasks.pop()
	if task == nil {
		return
	}

	s.resume()
	s.arrival(task)
}

func (s *sched) arrival(t *task) {
	// record running tasks
	// note that this must be placed in arrival because
	// s.cache may be early closed and then unlock will fail to delete lock.
	atomic.AddUint64(&s.running, 1)
	defer atomic.AddUint64(&s.running, ^uint64(0)) // -1

	ok, err := s.lock(t.Value)
	if err != nil || !ok {
		return
	}

	s.execute(t)
	// no need to hadle unlock fail since there is a timeout on cache
	// note that we must guarantee the task will never be arrival if the lock is not released.
	// refer to the s.running in above.
	s.unlock(t.Value)
}

func (s *sched) verify(t Task) (*time.Time, error) {
	r := &record{ID: t.GetID()}
	if err := r.read(); err != nil {
		return nil, err
	}
	taskTime := t.GetExecution()
	if taskTime.Before(r.Execution) {
		return &r.Execution, nil
	}
	return &taskTime, nil
}

func (s *sched) reschedule(t *task, when time.Time) {
	t.Value.SetExecution(when)
	// If save() is fail because of cache failure,
	// directly schedule the task without any hesitate
	// In case it may leads a problem of unabiding task
	// scheduling
	save(t.Value)

	s.pause()
	s.tasks.pushBack(t)
	s.resume()
}

func (s *sched) execute(t *task) {
	defer func() {
		if r := recover(); r != nil {
			t.future.put(fmt.Errorf("sched: task %s panic while executing, reason: %v", t.Value.GetID(), r))
		}
	}()

	execution, err := s.verify(t.Value)
	if err != nil {
		return
	}
	// for timer tollerance
	if execution.After(time.Now().UTC()) {
		// reschedule task, we must save the task again by using s.Setup
		s.reschedule(t, *execution)
		return
	}
	result, retry, err := t.Value.Execute()
	if retry || err != nil {
		s.reschedule(t, t.Value.GetRetryTime())
		return
	}
	if result == nil {
		result = fmt.Sprintf("sched: task %s success with nil return.", t.Value.GetID())
	}
	t.future.put(result)
	// NOTE: Generally this is not able to be fail.
	// However it may caused by the lost of connection.
	// Though task will be recovered when app restarts,
	// but it may leads an inconsistency, we need other
	// means to solve this problem
	s.del(t.Value)
}

// sched prefix for records
const (
	prefixTask = "sched:task:"
	prefixLock = "sched:lock:"
)

// save record into data store
func save(t Task) error {
	r := &record{
		ID:        t.GetID(),
		Execution: t.GetExecution(),
		Data:      t,
	}
	return r.save()
}

// del deletes record by id
func (s *sched) del(t Task) {
	s.cache.Del(prefixTask + t.GetID())
}

// lock the given task
func (s *sched) lock(t Task) (bool, error) {
	return s.cache.SetNX(prefixLock+t.GetID(), "locking", t.GetTimeout())
}

// unlock the given task explicitly
func (s *sched) unlock(t Task) {
	s.cache.Del(prefixLock + t.GetID())
}

// record of a schedule
type record struct {
	ID        string      `json:"id"`
	Execution time.Time   `json:"execution"`
	Data      interface{} `json:"data"`
}

// getRecords all records keys
func getRecords() (keys []string, err error) {
	keys, err = sched0.cache.Keys(prefixTask)
	ids := []string{}
	for _, key := range keys {
		ids = append(ids, strings.TrimPrefix(key, prefixTask))
	}
	return ids, err
}

// Read record with specified ID
func (r *record) read() (err error) {
	reply, err := sched0.cache.Get(prefixTask + r.ID)
	if err != nil {
		return
	}
	err = json.Unmarshal([]byte(reply), r)
	return
}

// Save record into data store
func (r *record) save() (err error) {
	data, err := json.Marshal(r)
	if err != nil {
		return
	}
	err = sched0.cache.Set(prefixTask+r.ID, string(data))
	return
}
