package pq

import (
	"container/heap"
	"sync"

	"github.com/changkun/goscheduler/task"
)

// TimerTaskQueue .
type TimerTaskQueue struct {
	heap   *itemHeap
	lookup map[string]*Item
	mu     sync.Mutex
}

// NewTimerTaskQueue .
func NewTimerTaskQueue() *TimerTaskQueue {
	pq := &itemHeap{}
	heap.Init(pq) // O(1) due to empty queue
	return &TimerTaskQueue{
		heap:   pq,
		lookup: map[string]*Item{},
	}
}

// Len of queue
func (m *TimerTaskQueue) Len() int {
	m.mu.Lock()
	defer m.mu.Unlock()

	length := m.heap.Len()
	return length
}

// Push item
func (m *TimerTaskQueue) Push(t task.Interface) {
	m.mu.Lock()
	defer m.mu.Unlock()

	_, ok := m.lookup[t.GetID()] // O(1) on average
	if ok {
		return
	}
	item := NewItem(t)
	heap.Push(m.heap, item)    // O(log(n))
	m.lookup[t.GetID()] = item // O(1)
}

// Pop item
func (m *TimerTaskQueue) Pop() task.Interface {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.heap.Len() == 0 {
		return nil
	}

	item := heap.Pop(m.heap).(*Item)     // O(log(n))
	delete(m.lookup, item.Value.GetID()) // O(1) on average
	return item.Value
}

// Peek the top priority item without deletion
func (m *TimerTaskQueue) Peek() task.Interface {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.heap.Len() == 0 {
		return nil
	}

	item := m.heap.Pop()      // O(log(n))
	m.heap.Push(item.(*Item)) // O(log(n))
	return item.(*Item).Value
}

// Update of a given task
func (m *TimerTaskQueue) Update(t task.Interface) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	item, ok := m.lookup[t.GetID()]
	if !ok {
		return false
	}

	item.priority = t.GetExecution()
	m.heap.Fix(item) // O(log(n))
	return true
}
