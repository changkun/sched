// Copyright 2018 Changkun Ou. All rights reserved.
// Use of this source code is governed by a MIT
// license that can be found in the LICENSE file.

package sched

import (
	"fmt"
	"math/rand/v2"
	"slices"
	"sync"
	"testing"
	"time"
)

func newEntry(id string, when time.Time) *entry {
	return &entry{task: newTask(id, when), when: when, futures: []*future{newFuture()}}
}

func TestQueueOrdersByTime(t *testing.T) {
	q := newQueue()
	if q.len() != 0 || q.peek() != nil || q.pop() != nil {
		t.Fatal("a new queue must be empty")
	}

	base := time.Now().UTC()
	order := rand.Perm(64)
	for _, i := range order {
		q.push(newEntry(fmt.Sprintf("q-%d", i), base.Add(time.Duration(i)*time.Second)))
	}
	if q.len() != 64 {
		t.Fatalf("len = %d, want 64", q.len())
	}

	var got []string
	for e := q.pop(); e != nil; e = q.pop() {
		got = append(got, e.task.ID())
	}
	want := make([]string, 64)
	for i := range want {
		want[i] = fmt.Sprintf("q-%d", i)
	}
	if !slices.Equal(got, want) {
		t.Fatalf("pop order = %v, want %v", got, want)
	}
}

func TestQueueUpdateMovesEntryBothWays(t *testing.T) {
	q := newQueue()
	base := time.Now().UTC()
	for i := range 8 {
		q.push(newEntry(fmt.Sprintf("q-%d", i), base.Add(time.Duration(i)*time.Second)))
	}

	// Move the last entry to the front, then the front entry to the back.
	last := q.find("q-7")
	if last == nil {
		t.Fatal("find must return a queued entry")
	}
	q.update(last, base.Add(-time.Hour))
	if got := q.peek().task.ID(); got != "q-7" {
		t.Fatalf("head = %s, want q-7", got)
	}
	q.update(last, base.Add(time.Hour))
	if got := q.peek().task.ID(); got != "q-0" {
		t.Fatalf("head = %s, want q-0", got)
	}

	if q.find("absent") != nil {
		t.Fatal("find of an unqueued identifier must return nil")
	}
	q.pop()
	if q.find("q-0") != nil {
		t.Fatal("a popped entry must leave the index")
	}
}

func TestEntryFansOutToEveryFuture(t *testing.T) {
	e := newEntry("fan", time.Now().UTC())
	e.futures = append(e.futures, newFuture(), newFuture())
	e.put("result")
	for i, f := range e.futures {
		if got := f.Get(); got != "result" {
			t.Fatalf("future %d = %v, want \"result\"", i, got)
		}
	}
}

func TestIntakeKeepsOrder(t *testing.T) {
	q := newIntake()
	if _, ok := q.pop(); ok {
		t.Fatal("an empty intake must report no entry")
	}

	base := time.Now().UTC()
	for i := range 32 {
		q.push(newEntry(fmt.Sprintf("i-%d", i), base))
	}
	for i := range 32 {
		e, ok := q.pop()
		if !ok {
			t.Fatalf("entry %d lost", i)
		}
		if got, want := e.task.ID(), fmt.Sprintf("i-%d", i); got != want {
			t.Fatalf("pop = %s, want %s", got, want)
		}
	}
	if _, ok := q.pop(); ok {
		t.Fatal("intake must be empty again")
	}
}

func TestIntakeAcceptsConcurrentProducers(t *testing.T) {
	q := newIntake()
	const producers, each = 16, 64

	var wg sync.WaitGroup
	for p := range producers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range each {
				q.push(newEntry(fmt.Sprintf("p-%d-%d", p, i), time.Now().UTC()))
			}
		}()
	}

	seen := map[string]bool{}
	done := make(chan struct{})
	go func() { wg.Wait(); close(done) }()
	for len(seen) < producers*each {
		e, ok := q.pop()
		if !ok {
			select {
			case <-done:
				// Producers finished; one more drain settles it.
				for e, ok := q.pop(); ok; e, ok = q.pop() {
					seen[e.task.ID()] = true
				}
				if len(seen) < producers*each {
					t.Fatalf("intake lost %d entries", producers*each-len(seen))
				}
			default:
			}
			continue
		}
		seen[e.task.ID()] = true
	}
}

func TestBroadcastWakesEveryWaiter(t *testing.T) {
	b := newBroadcast()
	var wg sync.WaitGroup
	for range 8 {
		wg.Add(1)
		ch := b.wait()
		go func() { defer wg.Done(); <-ch }()
	}
	b.fire()

	done := make(chan struct{})
	go func() { wg.Wait(); close(done) }()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("fire must wake every waiter")
	}

	// A fresh generation replaces the closed one.
	select {
	case <-b.wait():
		t.Fatal("the new generation must stay open")
	default:
	}
}
