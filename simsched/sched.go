// Copyright 2018 Changkun Ou. All rights reserved.
// Use of this source code is governed by a MIT
// license that can be found in the LICENSE file.

package simsched

import (
	"container/heap"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"
)

// Task interface for sched
type Task interface {
	// GetID must returns a unique ID for all of the scheduled task.
	GetID() (id string)
	// SetID will set id as the unique ID for the scheduled task.
	SetID(id string)
	// GetExecution returns the time for task execution.
	GetExecution() (execute time.Time)
	// SetExecution sets a new time for the task execution
	SetExecution(new time.Time) (old time.Time)
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
	// pause sched0 fisrt.
	Pause()

	// wait until all started tasks (i.e. tasks is executing other than timing) stops
	//
	// note that the following busy wait satisfies sequential consistency
	// memory model since the loop does not wait any value but only checkes
	// sched0.running atomically.
	running := atomic.LoadUint64(&sched0.running)
	for {
		current := atomic.LoadUint64(&sched0.running)
		if current < running {
			running = current
		}
		// if running descreased to 0 then sched is actually can be terminated
		if running == 0 {
			break
		}
		// use runtime.Gosched vacates CPU for other goroutines
		// instead of spin loop
		runtime.Gosched()
	}

	if v := atomic.AddUint64(&sched0.pausing, ^uint64(0)); v != 0 {
		panic(fmt.Sprintf("sched: stop is wrongly implemented: %v", v))
	}
}

// Wait waits all tasks to be scheduled.
func Wait() {
	for sched0.tasks.length() != 0 {
		runtime.Gosched()
	}
}

// Submit given tasks
func Submit(t Task) TaskFuture {
	return sched0.submit(t)
}

// Trigger given tasks immediately
func Trigger(t Task) TaskFuture {
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

var sched0 = &sched{
	timer: unsafe.Pointer(time.NewTimer(0)),
	tasks: newTaskQueue(),
}

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

// submit given tasks
func (s *sched) submit(t Task) (future TaskFuture) {
	future = s.schedule(t)
	return
}

// trigger given tasks immediately
func (s *sched) trigger(t Task) TaskFuture {
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
	// newt := time.NewTimer(duration)
	for {
		old := atomic.LoadPointer(&s.timer)
		if atomic.CompareAndSwapPointer(&s.timer, old, unsafe.Pointer(time.NewTimer(d))) {
			tt := (*time.Timer)(old)
			if tt != nil {
				if tt.Stop() {
					tt.Reset(d)
					atomic.StorePointer(&s.timer, unsafe.Pointer(tt))
				}
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
	t := s.tasks.peek()
	if t == nil {
		return
	}
	s.setTimer(t.GetExecution().Sub(time.Now().UTC()))
	go s.timing()
}

func (s *sched) timing() {
	timer := s.getTimer()
	if timer == nil {
		return
	}
	<-timer.C
	s.worker()
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
	atomic.AddUint64(&s.running, 1)
	s.execute(t)
	atomic.AddUint64(&s.running, ^uint64(0)) // -1
}

func (s *sched) reschedule(t *task, when time.Time) {
	t.Value.SetExecution(when)
	s.pause()
	s.tasks.pushBack(t)
	s.resume()
}

func (s *sched) execute(t *task) {
	defer func() {
		if r := recover(); r != nil {
			t.future.write(fmt.Sprintf("sched: task %s panic while executing, reason: %v", t.Value.GetID(), r))
		}
	}()

	// for timer tollerance
	if t.Value.GetExecution().After(time.Now().UTC()) {
		// reschedule task, we must save the task again by using s.Setup
		s.reschedule(t, t.Value.GetExecution())
		return
	}
	result, retry, err := t.Value.Execute()
	if retry || err != nil {
		s.reschedule(t, t.Value.GetRetryTime())
		return
	}
	if result == nil {
		result = fmt.Sprintf("sched: task %s success with nil return", t.Value.GetID())
	}
	t.future.write(result)
}

// TaskQueue implements a timer queue based on a heap
// Its supports bi-direction accessing, such as access value by key
// or access key by its value
//
//                Time Complexity      Space Complexity
//   New()              O(1)                 O(1)
//   Len()              O(1)                 O(1)
//   Push()   amortized O(log(n))            O(1)
//   Pop()    amortized O(log(n))            O(1)
//   Peek()   amortized O(log(n))            O(1)
//   Update() amortized O(log(n))            O(1)
//
// Total space complexity: O(n + n) where n = queue.Len(), which is
// a slice + a loopup hash table(map).
//
// Worst case for "amortized" is O(n)
//
// Lock-free priority queue is possible. However, is it possible
// to implement in pq with lookup? we cloud not find literature indication yet.
type taskQueue struct {
	heap   *taskHeap
	lookup map[string]*task
	mu     sync.Mutex
}

func newTaskQueue() *taskQueue {
	pq := &taskHeap{}
	heap.Init(pq)
	return &taskQueue{
		heap:   pq, // O(1) due to empty queue
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
func (m *taskQueue) push(t Task) (*future, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	old, ok := m.lookup[t.GetID()] // O(1) amortized
	if ok {
		return old.future, false
	}
	item := newTaskItem(t)
	heap.Push(m.heap, item)    // O(log(n))
	m.lookup[t.GetID()] = item // O(1)
	return item.future, true
}

func (m *taskQueue) pushBack(t *task) {
	m.mu.Lock()
	defer m.mu.Unlock()

	heap.Push(m.heap, t)          // O(log(n))
	m.lookup[t.Value.GetID()] = t // O(1)
}

// Pop item
func (m *taskQueue) pop() *task {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.heap.Len() == 0 {
		return nil
	}

	item := heap.Pop(m.heap).(*task)     // O(log(n))
	delete(m.lookup, item.Value.GetID()) // O(1) amortized
	return item
}

// peek the top priority item without deletion
func (m *taskQueue) peek() Task {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.heap.Len() == 0 {
		return nil
	}
	return (*m.heap)[0].Value
}

// update of a given task
func (m *taskQueue) update(t Task) (*future, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	item, ok := m.lookup[t.GetID()]
	if !ok {
		return nil, false
	}

	item.priority = t.GetExecution()
	item.Value = t
	heap.Fix(m.heap, item.index) // O(log(n))
	return item.future, true
}

// a task is something we manage in a priority queue.
type task struct {
	Value Task // for storage

	// The index is needed by update and is maintained by the heap.Interface methods.
	index    int       // The index of the item in the heap.
	priority time.Time // type of time for priority
	future   *future
}

// NewTaskItem creates a new queue item
func newTaskItem(t Task) *task {
	return &task{
		Value:    t,
		priority: t.GetExecution(),
		future:   &future{},
	}
}

type future struct {
	value atomic.Value
}

// Get implements TaskFuture interface
func (f *future) Get() (v interface{}) {
	// spin until value is stored in future.value
	for {
		if v = f.value.Load(); v != nil {
			return
		}
		runtime.Gosched()
	}
}

func (f *future) write(v interface{}) {
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
