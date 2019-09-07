// Copyright 2018-2019 Changkun Ou. All rights reserved.
// Use of this source code is governed by a MIT
// license that can be found in the LICENSE file.

package sched

import (
	"container/heap"
	"sync"
	"sync/atomic"
	"time"
)

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
func (m *taskQueue) length() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.heap.Len()
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

	item := heap.Pop(m.heap).(*task) // O(log(n))
	heap.Push(m.heap, item)          // O(log(n))
	return item.Value
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
