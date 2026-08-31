// Copyright 2018 Changkun Ou. All rights reserved.
// Use of this source code is governed by a MIT
// license that can be found in the LICENSE file.

package sched

import "sync/atomic"

// intake is a multi-producer, single-consumer queue that carries submitted
// entries from callers to the scheduler loop.
//
// push is wait-free: it runs one atomic exchange and one atomic store, with
// no loop and no retry, so a caller finishes in a bounded number of steps no
// matter what any other goroutine does. This is the property that lets the
// public API drop the mutex the previous design used to guard the task heap.
//
// The design is Vyukov's intrusive MPSC queue. A producer exchanges the tail
// with its new node and then links the previous tail to it. Between those two
// stores the queue is briefly disconnected, and pop reports "empty" although
// the node exists. sched.submit signals the loop only after the link store,
// so every node that pop can miss gets its own wake-up afterwards and none is
// ever lost.
type intake struct {
	tail atomic.Pointer[node] // producers
	head *node                // consumer only
}

type node struct {
	next  atomic.Pointer[node]
	value *entry
}

func newIntake() *intake {
	stub := &node{}
	q := &intake{head: stub}
	q.tail.Store(stub)
	return q
}

// push appends e. Safe for any number of concurrent producers, wait-free.
func (q *intake) push(e *entry) {
	n := &node{value: e}
	prev := q.tail.Swap(n)
	prev.next.Store(n)
}

// pop removes the oldest entry. Only the consumer goroutine may call it. It
// reports false when the queue is empty and when a concurrent push has not
// linked its node yet.
func (q *intake) pop() (*entry, bool) {
	next := q.head.next.Load()
	if next == nil {
		return nil, false
	}
	e := next.value
	next.value = nil // release the entry for collection
	q.head = next
	return e, true
}
