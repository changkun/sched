// Copyright 2018 Changkun Ou. All rights reserved.
// Use of this source code is governed by a MIT
// license that can be found in the LICENSE file.

package sched

import (
	"encoding/json"
	"log"
	"reflect"
	"runtime"
	"strings"
	"sync/atomic"
	"time"
	"unsafe"
)

// Init initialize task scheduler
func Init(db string, all ...Task) {
	connectCache(db)
	sched0 = &sched{
		timer: unsafe.Pointer(time.NewTimer(0)),
		tasks: newTaskQueue(),
	}
	if err := sched0.recover(all...); err != nil {
		log.Fatalf("sched: recover tasks failed, error: %v", err)
	}
	log.Printf("sched: runtime scheduler has initialized and up for scheduling.")
}

// Stop stops runtime scheduler gracefully.
// Note that the call should only be called then application terminates
func Stop() {
	// pause sched0 fisrt.
	Pause()

	// wait until all started tasks (i.e. tasks is executing other than timing) stops
	//
	// note that the following busy wait satisfies sequential consistency
	// memory model since the loop does not wait any value but only checkes
	// sched0.running atomically.
	running := atomic.LoadUint64(&sched0.running)
	log.Printf("sched: waiting for sched to stop, running tasks: %d", running)
	for {
		current := atomic.LoadUint64(&sched0.running)
		if current < running {
			running = current
			log.Printf("sched: running tasks are reduced to %d", running)
		}
		// if running descreased to 0 then sched is actually can be terminated
		if running == 0 {
			return
		}
		// use runtime.Gosched vacates CPU for other goroutines
		// instead of spin loop
		runtime.Gosched()
	}
}

// Submit given tasks
func Submit(ts ...Task) error {
	return sched0.submit(ts...)
}

// Trigger given tasks immediately
func Trigger(ts ...Task) error {
	return sched0.trigger(ts...)
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
//
// sched uses greedy scheduling algorithm that creates many goroutines
// at the same time if and only if tasks need be executed at the same time,
// otherwise there will be only one goroutine for executing the task.
//
// Moreover, there will be no goroutine if the task queue is empty,
// which makes the approach better than immortal channel loop.
//
// The performance, in sched, is always better than an immportal channel loop
// since lock costs satisfy big lock (chan) > standard lock (mutex) > atomic (sched).
//
// Please note that there is still an optimization trick for sched,
// the task queue is implemented via mutex, which makes the task queue
// much slower than lock-free (spin lock with cas algorithm), therefore
// future optimization could consider to implement a lock-free priority queue
// for the timer task scheduling.
type sched struct {
	// running counts the tasks already starts that cannot be stopped,
	// for timing tasks that still waiting for execution, call sched.tasks.Len().
	running uint64 // atomic
	// pausing is a sign that indicates if sched should stop running.
	pausing uint64 // atomic
	// timer is the only timer during the runtime
	timer unsafe.Pointer // *time.Timer
	// tasks is a TaskQueue that stores all unscheduled tasks in memory
	tasks *taskQueue
}

func (s *sched) recover(ts ...Task) error {
	ids, err := getRecords()
	if err != nil {
		return err
	}

	for _, t := range ts {
		for i := range ids {
			r := &record{ID: ids[i]}
			if err := r.read(); err != nil {
				return err
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
			if err := json.Unmarshal(data, &temp2); err != nil {
				continue
			}
			if reflect.DeepEqual(temp1, temp2) || temp2 == nil || !temp2.IsValidID() {
				continue
			}
			temp2.SetID(ids[i])
			temp2.SetExecution(r.Execution)
			s.tasks.Push(temp2)
		}
	}
	s.resume()
	return nil
}

// submit given tasks
func (s *sched) submit(ts ...Task) error {
	for i := range ts {
		// save asap
		if err := save(ts[i]); err != nil {
			return err
		}
		s.schedule(ts[i])
	}
	return nil
}

// trigger given tasks immediately
func (s *sched) trigger(ts ...Task) error {
	for i := range ts {
		ts[i].SetExecution(time.Now().UTC())
		if err := s.submit(ts[i]); err != nil {
			return err
		}
	}
	return nil
}

func (s *sched) schedule(t Task) {
	s.pause()
	defer s.resume()

	// if priority is able to be update
	if s.tasks.Update(t) {
		log.Printf("sched: existing task %s has been updated.", t.GetID())
		return
	}

	s.tasks.Push(t)
}

func (s *sched) setTimer(duration time.Duration) {
	// spin lock
	for {
		old := atomic.LoadPointer(&s.timer)
		if atomic.CompareAndSwapPointer(&s.timer, old, unsafe.Pointer(time.NewTimer(duration))) {
			if (*time.Timer)(old) != nil {
				(*time.Timer)(old).Stop()
			}
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
	// fast path.
	// this check is necessary, sometimes timer will become zero value.
	if (*time.Timer)(atomic.LoadPointer(&s.timer)) == nil {
		return
	}

	// spin lock
	for {
		old := atomic.LoadPointer(&s.timer)
		if atomic.CompareAndSwapPointer(&s.timer, old, nil) {
			if (*time.Timer)(old) != nil {
				(*time.Timer)(old).Stop()
			}
			return
		}
	}
}

func (s *sched) ispausing() bool {
	return atomic.LoadUint64(&s.pausing) > 0
}

func (s *sched) resume() {
	t := s.tasks.Peek()
	if t == nil {
		return
	}
	s.setTimer(t.GetExecution().Sub(time.Now().UTC()))
	go func() {
		timer := s.getTimer()
		if timer == nil {
			return
		}
		<-timer.C

		// fast path.
		// if sched requires pausing, then stop executing and resume it.
		if s.ispausing() {
			return
		}

		// medium path.
		// stop execution if task queue is empty
		t := s.tasks.Pop()
		if t == nil {
			return
		}

		s.resume()
		s.arrival(t)
	}()
}

func (s *sched) arrival(t Task) {
	ok, err := lock(t)
	if err != nil || !ok {
		return
	}

	s.execute(t)
	// no need to hadle unlock fail
	// since there is a timeout on cache
	unlock(t)
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

func (s *sched) retry(t Task) {
	t.SetExecution(t.GetRetryTime())
	if err := s.submit(t); err != nil {
		// If submit() is fail because of cache failure,
		// directly schedule the task without any hesitate
		// In case it may leads a problem of unabiding task
		// scheduling
		s.schedule(t)
	}
}

func (s *sched) execute(t Task) {
	// record running tasks
	atomic.AddUint64(&s.running, 1)
	defer atomic.AddUint64(&s.running, ^uint64(0)) // -1
	defer func() {
		if r := recover(); r != nil {
			log.Printf("sched: task %s panic while executing, reason: %v", t.GetID(), r)
		}
	}()

	execution, err := s.verify(t)
	if err != nil {
		return
	}
	// for timer tollerance
	if execution.After(time.Now().UTC()) {
		// reschedule task, we must save the task again by using s.Setup
		t.SetExecution(*execution)
		s.submit(t)
		return
	}
	retry, err := t.Execute()
	if retry || err != nil {
		s.retry(t)
		return
	}

	// NOTE: Generally this is not able to be fail.
	// However it may caused by the lost of connection.
	// Though task will be recovered when app restarts,
	// but it may leads an inconsistency, we need other
	// means to solve this problem
	del(t)
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
func del(t Task) {
	cache0.DEL(prefixTask + t.GetID())
}

// lock the given task
func lock(t Task) (bool, error) {
	return cache0.SETNX(prefixLock+t.GetID(), "locking", t.GetTimeout())
}

// unlock the given task explicitly
func unlock(t Task) {
	cache0.DEL(prefixLock + t.GetID())
}

// record of a schedule
type record struct {
	ID        string      `json:"id"`
	Execution time.Time   `json:"execution"`
	Data      interface{} `json:"data"`
}

// getRecords all records keys
func getRecords() (keys []string, err error) {
	keys, err = cache0.KEYS(prefixTask)
	ids := []string{}
	for _, key := range keys {
		ids = append(ids, strings.TrimPrefix(key, prefixTask))
	}
	return ids, err
}

// Read record with specified ID
func (r *record) read() (err error) {
	reply, err := cache0.GET(prefixTask + r.ID)
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
	err = cache0.SET(prefixTask+r.ID, string(data))
	return
}
