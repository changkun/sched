// Copyright 2018 Changkun Ou. All rights reserved.
// Use of this source code is governed by a MIT
// license that can be found in the LICENSE file.

package pq

import (
	"container/heap"
	"sync"
	"sync/atomic"
	"unsafe"

	"github.com/changkun/sched/task"
)

// TaskQueue implements a timer queue based on a heap
// Its supports bi-direction accessing, such as access value by key
// or access key by its value
//
//               Time Complexity      Space Complexity
//   New()             O(1)                 O(1)
//   Len()             O(1)                 O(1)
//   Push()    O(log(n)) on average         O(1)
//   Pop()     O(log(n)) on average         O(1)
//   Peek()        O(log(n))                O(1)
//   Update()  O(log(n)) on average         O(1)
//
// Total space complexity: O(n + n) where n = queue.Len(), which is
// a slice + a loopup hash table(map).
//
// Worst case for "on average" is O(n)
type TaskQueue struct {
	heap   unsafe.Pointer //*itemHeap
	lookup map[string]*Item
	mu     sync.Mutex
}

// NewTaskQueue .
func NewTaskQueue() *TaskQueue {
	pq := &itemHeap{}
	heap.Init(pq) // O(1) due to empty queue
	return &TaskQueue{
		heap:   unsafe.Pointer(pq),
		lookup: map[string]*Item{},
	}
}

// Len of queue
func (m *TaskQueue) Len() int {
	return ((*itemHeap)(atomic.LoadPointer(&m.heap))).Len()
}

// Push item
func (m *TaskQueue) Push(t task.Interface) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	_, ok := m.lookup[t.GetID()] // O(1) on average
	if ok {
		return false
	}
	item := NewItem(t)
	heap.Push((*itemHeap)(m.heap), item) // O(log(n))
	m.lookup[t.GetID()] = item           // O(1)
	return true
}

// Pop item
func (m *TaskQueue) Pop() task.Interface {
	m.mu.Lock()
	defer m.mu.Unlock()

	if (*itemHeap)(m.heap).Len() == 0 {
		return nil
	}

	item := heap.Pop((*itemHeap)(m.heap)).(*Item) // O(log(n))
	delete(m.lookup, item.Value.GetID())          // O(1) on average
	return item.Value
}

// Peek the top priority item without deletion
func (m *TaskQueue) Peek() task.Interface {
	m.mu.Lock()
	defer m.mu.Unlock()

	if (*itemHeap)(m.heap).Len() == 0 {
		return nil
	}

	item := heap.Pop((*itemHeap)(m.heap)).(*Item) // O(log(n))
	heap.Push((*itemHeap)(m.heap), item)          // O(log(n))
	return item.Value
}

// Update of a given task
func (m *TaskQueue) Update(t task.Interface) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	item, ok := m.lookup[t.GetID()]
	if !ok {
		return false
	}

	item.priority = t.GetExecution()
	item.Value = t
	heap.Fix((*itemHeap)(m.heap), item.index) // O(log(n))
	return true
}
