// Copyright 2019 Changkun Ou. All rights reserved.
// Use of this source code is governed by a MIT
// license that can be found in the LICENSE file.

package simsched

import (
	"container/heap"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"
)

// Task interface for sched
type Task interface {
	// GetID must returns a unique ID for all of the scheduled task.
	GetID() (id string)
	// GetExecution returns the time for task execution.
	GetExecution() (execute time.Time)
	// GetRetryTime returns the retry time if a task was failed.
	GetRetryTime() (execute time.Time)
	// Execute executes the actual task, it can return a result,
	// or if the task need a retry, or it was failed in this execution.
	Execute() (result interface{}, retry bool, fail error)
}

// TaskFuture is the future of Task execution
type TaskFuture interface {
	Get() interface{}
}

// Stop stops runtime scheduler gracefully.
// Note that the call should only be called then application terminates
func Stop() {
	Pause()

	// wait until all started tasks (i.e. tasks is executing other than
	// timing) stops
	for atomic.LoadUint64(&sched0.running) > 0 {
	}

	// reset pausing indicator
	atomic.AddUint64(&sched0.pausing, ^uint64(0))
}

// Wait waits all tasks to be scheduled.
func Wait() {
	// With function call, no need for runtime.Gosched()
	for sched0.tasks.length() != 0 {
	}
}

// Submit given tasks
func Submit(t Task) TaskFuture {
	return sched0.schedule(t, t.GetExecution())
}

// Trigger given tasks immediately
func Trigger(t Task) TaskFuture {
	return sched0.schedule(t, time.Now().UTC())
}

// Pause stops the sched timing
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

var sched0 = &sched{
	timer: unsafe.Pointer(time.NewTimer(0)),
	tasks: newTaskQueue(),
}

// sched is the actual scheduler for task scheduling
//
// sched implements greedy scheduling, with a timer and a task queue,
// the task queue is a priority queue that orders tasks by executing
// time. The timer is the only time.Timer lives in runtime, it serves
// the head task in the task queue.
type sched struct {
	// running counts the tasks already starts that cannot be stopped.
	running uint64 // atomic
	// pausing is a sign that indicates if sched should stop running.
	pausing uint64 // atomic
	// timer is the only timer during the runtime
	timer unsafe.Pointer // *time.Timer
	// tasks is a TaskQueue that stores all unscheduled tasks in memory
	tasks *taskQueue
}

func (s *sched) schedule(t Task, when time.Time) TaskFuture {
	s.pause()

	// if priority is able to be update
	if future, ok := s.tasks.update(t, when); ok {
		s.resume()
		return future
	}

	future := s.tasks.push(newTaskItem(t, when))
	s.resume()
	return future
}

func (s *sched) reschedule(t *task, when time.Time) {
	s.pause()
	t.priority = when
	s.tasks.push(t)
	s.resume()
}

func (s *sched) getTimer() *time.Timer {
	return (*time.Timer)(atomic.LoadPointer(&s.timer))
}

func (s *sched) setTimer(d time.Duration) {
	for {
		// fast path: reuse the timer
		old := atomic.LoadPointer(&s.timer)
		if (*time.Timer)(old).Stop() {
			(*time.Timer)(old).Reset(d)
			return
		}

		// slow path: fail to stop, use a new timer.
		// this happens only if the sched is super busy.
		if atomic.CompareAndSwapPointer(&s.timer, old,
			unsafe.Pointer(time.NewTimer(d))) {
			(*time.Timer)(old).Stop()
			return
		}
	}
}

// pause pauses sched without pause tasks from running
func (s *sched) pause() {
	(*time.Timer)(atomic.LoadPointer(&s.timer)).Stop()
}

func (s *sched) resume() {
	t := s.tasks.peek()
	if t == nil {
		return
	}
	s.setTimer(t.GetExecution().Sub(time.Now().UTC()))

	// TODO: reuse goroutine here
	go func() {
		<-s.getTimer().C
		s.worker()
	}()
}

func (s *sched) worker() {
	// fast path.
	// if sched requires pausing, then stop executing and resume it.
	if atomic.LoadUint64(&s.pausing) > 0 {
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
	atomic.AddUint64(&s.running, 1)
	s.execute(t)
	atomic.AddUint64(&s.running, ^uint64(0)) // -1
}

func (s *sched) execute(t *task) {
	defer func() {
		if r := recover(); r != nil {
			t.future.put(
				fmt.Errorf(
					"sched: task %s panic while executing, reason: %v",
					t.value.GetID(), r))
		}
	}()

	// for timer tollerance
	if t.value.GetExecution().After(time.Now().UTC()) {
		// reschedule task, we must save the task again by using s.Setup
		s.reschedule(t, t.value.GetExecution())
		return
	}
	result, retry, err := t.value.Execute()
	if retry || err != nil {
		s.reschedule(t, t.value.GetRetryTime())
		return
	}
	// avoid nil result
	if result == nil {
		result = fmt.Sprintf("sched: task %s success with nil return",
			t.value.GetID())
	}
	t.future.put(result)
}

// TaskQueue implements a timer queue based on a heap
// Its supports bi-direction accessing, such as access value by key
// or access key by its value
//
// TODO: lock-free
type taskQueue struct {
	heap   *taskHeap
	lookup map[string]*task
	mu     sync.Mutex
}

func newTaskQueue() *taskQueue {
	return &taskQueue{
		heap:   &taskHeap{},
		lookup: map[string]*task{},
	}
}

// length of queue
func (m *taskQueue) length() (l int) {
	m.mu.Lock()
	l = m.heap.Len()
	m.mu.Unlock()
	return
}

// push item
func (m *taskQueue) push(t *task) *future {
	m.mu.Lock()

	old, ok := m.lookup[t.value.GetID()] // O(1) amortized
	if ok {
		m.mu.Unlock()
		return old.future
	}
	heap.Push(m.heap, t)          // O(log(n))
	m.lookup[t.value.GetID()] = t // O(1)
	m.mu.Unlock()
	return t.future
}

// Pop item
func (m *taskQueue) pop() *task {
	m.mu.Lock()

	if m.heap.Len() == 0 {
		m.mu.Unlock()
		return nil
	}

	item := heap.Pop(m.heap).(*task)     // O(log(n))
	delete(m.lookup, item.value.GetID()) // O(1) amortized
	m.mu.Unlock()
	return item
}

// peek the top priority item without deletion
func (m *taskQueue) peek() (t Task) {
	m.mu.Lock()

	if m.heap.Len() == 0 {
		m.mu.Unlock()
		return nil
	}
	t = (*m.heap)[0].value
	m.mu.Unlock()
	return
}

// update of a given task
func (m *taskQueue) update(t Task, when time.Time) (*future, bool) {
	m.mu.Lock()
	item, ok := m.lookup[t.GetID()]
	if !ok {
		m.mu.Unlock()
		return nil, false
	}

	item.priority = when
	item.value = t
	heap.Fix(m.heap, item.index) // O(log(n))
	m.mu.Unlock()
	return item.future, true
}

// a task is something we manage in a priority queue.
type task struct {
	value Task // for storage

	// The index is needed by update and is maintained by the
	// heap.Interface methods.
	index    int       // The index of the item in the heap.
	priority time.Time // type of time for priority
	future   *future
}

// NewTaskItem creates a new queue item
func newTaskItem(t Task, when time.Time) *task {
	return &task{value: t, priority: when, future: &future{}}
}

type future struct {
	value atomic.Value
}

// Get implements TaskFuture interface
func (f *future) Get() (v interface{}) {
	// spin until value is stored in future.value
	for ; v == nil; v = f.value.Load() {
	}
	return
}

func (f *future) put(v interface{}) {
	f.value.Store(v)
}

type taskHeap []*task

func (pq taskHeap) Len() int {
	return len(pq)
}

func (pq taskHeap) Less(i, j int) bool {
	return pq[i].priority.Before(pq[j].priority)
}

func (pq taskHeap) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].index = i
	pq[j].index = j
}

func (pq *taskHeap) Pop() interface{} {
	old := *pq
	n := len(old)
	item := old[n-1]
	*pq = old[0 : n-1]
	return item
}

func (pq *taskHeap) Push(x interface{}) {
	item := x.(*task)
	item.index = len(*pq)
	*pq = append(*pq, item)
}
