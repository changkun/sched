// Copyright 2018 Changkun Ou. All rights reserved.
// Use of this source code is governed by a MIT
// license that can be found in the LICENSE file.

package sched

import "sync/atomic"

// broadcast wakes every waiter without a lock and without losing a signal.
//
// A waiter takes the current generation channel and then rechecks its
// condition. fire swaps in a fresh generation and closes the old one, so a
// signal that arrives after the waiter took the channel still wakes it. The
// swap decides which goroutine closes which channel, so concurrent fires
// never close the same channel twice.
type broadcast struct {
	gen atomic.Pointer[chan struct{}]
}

func newBroadcast() *broadcast {
	b := &broadcast{}
	ch := make(chan struct{})
	b.gen.Store(&ch)
	return b
}

// wait returns the channel that the next fire closes.
func (b *broadcast) wait() <-chan struct{} { return *b.gen.Load() }

// fire wakes everyone that waits on the current generation.
func (b *broadcast) fire() {
	next := make(chan struct{})
	close(*b.gen.Swap(&next))
}
