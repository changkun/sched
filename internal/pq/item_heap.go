// Copyright 2018 Changkun Ou. All rights reserved.
// Use of this source code is governed by a MIT
// license that can be found in the LICENSE file.

package pq

import (
	"time"

	"github.com/changkun/sched/task"
)

// An Item is something we manage in a priority queue.
type Item struct {
	Value task.Interface // for storage

	// The index is needed by update and is maintained by the heap.Interface methods.
	index    int       // The index of the item in the heap.
	priority time.Time // type of time for priority
}

// NewItem creates a new queue item
func NewItem(t task.Interface) *Item {
	return &Item{
		Value:    t,
		priority: t.GetExecution(),
	}
}

type itemHeap []*Item

func (pq itemHeap) Len() int {
	return len(pq)
}

func (pq itemHeap) Less(i, j int) bool {
	return pq[i].priority.Before(pq[j].priority)
}

func (pq itemHeap) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].index = i
	pq[j].index = j
}

func (pq *itemHeap) Pop() interface{} {
	old := *pq
	n := len(old)
	item := old[n-1]
	*pq = old[0 : n-1]
	return item
}

func (pq *itemHeap) Push(x interface{}) {
	item := x.(*Item)
	item.index = len(*pq)
	*pq = append(*pq, item)
}
