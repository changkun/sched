// Copyright 2018 Changkun Ou. All rights reserved.
// Use of this source code is governed by a MIT
// license that can be found in the LICENSE file.

package sched

import "time"

// entry is one task inside the scheduler queue.
//
// futures holds every Future handed to a caller for this task. Submitting a
// task whose identifier is already queued does not duplicate the work; it
// appends a Future to the entry that is already there, and one result fans
// out to all of them.
type entry struct {
	task    Task
	when    time.Time
	index   int // position in queue.heap, maintained by the queue
	futures []*future
}

// put publishes v to every caller that waits for this task.
func (e *entry) put(v any) {
	for _, f := range e.futures {
		f.put(v)
	}
}

// queue is a min-heap of entries ordered by execution time, with an index
// from task identifier to entry.
//
//	           time          space
//	len        O(1)          O(1)
//	push       O(log n)      O(1)
//	pop        O(log n)      O(1)
//	peek       O(1)          O(1)
//	update     O(log n)      O(1)
//	remove     O(log n)      O(1)
//
// The queue carries no synchronization of its own. Exactly one goroutine,
// the scheduler loop in sched.run, ever touches it. That is what removes the
// mutex the previous design needed, and it is why every method here must
// stay unexported.
type queue struct {
	heap   []*entry
	lookup map[string]*entry
}

func newQueue() *queue {
	return &queue{lookup: map[string]*entry{}}
}

func (q *queue) len() int { return len(q.heap) }

// peek returns the entry that runs next, or nil if the queue is empty.
func (q *queue) peek() *entry {
	if len(q.heap) == 0 {
		return nil
	}
	return q.heap[0]
}

// find returns the queued entry for id, or nil.
func (q *queue) find(id string) *entry { return q.lookup[id] }

// push inserts e. The caller must have checked that e.task.ID() is not
// queued yet.
func (q *queue) push(e *entry) {
	e.index = len(q.heap)
	q.heap = append(q.heap, e)
	q.up(e.index)
	q.lookup[e.task.ID()] = e
}

// pop removes and returns the entry that runs next, or nil if empty.
func (q *queue) pop() *entry {
	if len(q.heap) == 0 {
		return nil
	}
	e := q.heap[0]
	q.swap(0, len(q.heap)-1)
	q.heap[len(q.heap)-1] = nil
	q.heap = q.heap[:len(q.heap)-1]
	if len(q.heap) > 0 {
		q.down(0)
	}
	delete(q.lookup, e.task.ID())
	e.index = -1
	return e
}

// update moves a queued entry to a new execution time.
func (q *queue) update(e *entry, when time.Time) {
	e.when = when
	i := e.index
	if !q.up(i) {
		q.down(i)
	}
}

func (q *queue) less(i, j int) bool { return q.heap[i].when.Before(q.heap[j].when) }

func (q *queue) swap(i, j int) {
	q.heap[i], q.heap[j] = q.heap[j], q.heap[i]
	q.heap[i].index = i
	q.heap[j].index = j
}

// up sifts the element at i towards the root and reports whether it moved.
func (q *queue) up(i int) bool {
	start := i
	for i > 0 {
		parent := (i - 1) / 2
		if !q.less(i, parent) {
			break
		}
		q.swap(i, parent)
		i = parent
	}
	return i != start
}

// down sifts the element at i towards the leaves.
func (q *queue) down(i int) {
	for {
		l, r := 2*i+1, 2*i+2
		small := i
		if l < len(q.heap) && q.less(l, small) {
			small = l
		}
		if r < len(q.heap) && q.less(r, small) {
			small = r
		}
		if small == i {
			return
		}
		q.swap(i, small)
		i = small
	}
}
