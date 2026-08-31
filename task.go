// Copyright 2018 Changkun Ou. All rights reserved.
// Use of this source code is governed by a MIT
// license that can be found in the LICENSE file.

package sched

import (
	"sync/atomic"
	"time"
)

// Task is the unit of work that sched schedules.
//
// An implementation must be a struct that json.Marshal can encode, because
// sched persists it to recover the schedule after a restart. Exported fields
// are the only state that survives a restart; sched calls SetID and
// SetExecution to restore the rest.
type Task interface {
	// ID returns an identifier that is unique among all scheduled tasks.
	ID() string
	// SetID restores the identifier of a recovered task.
	SetID(id string)
	// Execution returns the time at which the task must run.
	Execution() time.Time
	// SetExecution sets the time at which the task must run.
	SetExecution(t time.Time)
	// Timeout returns the lifetime of the distributed lock that keeps
	// replicas from running the task twice. sched releases the lock as
	// soon as Execute returns, so the lifetime only matters when the
	// replica dies while the task runs. Give it more than Execute
	// normally needs: a lifetime that expires during execution lets a
	// second replica start the same task, and one much longer than that
	// keeps the task unclaimable after a crash.
	Timeout() time.Duration
	// RetryTime returns the time of the next attempt if Execute asks for
	// a retry or fails.
	RetryTime() time.Time
	// Execute runs the task. It returns the result to publish through the
	// Future, whether the task must run again, and the failure if there
	// was one. A non-nil error or retry == true reschedules the task at
	// RetryTime.
	Execute() (result any, retry bool, err error)
}

// Future is the pending result of a scheduled task.
type Future interface {
	// Get blocks until the task completes and returns its result. If the
	// task panics, Get returns an error that describes the panic.
	Get() any
	// Done is closed when the result is available. Use it to wait with a
	// select statement, for example against a context.
	Done() <-chan struct{}
}

// future is the sched-side implementation of Future.
type future struct {
	done   chan struct{}
	filled atomic.Bool
	value  any // written before done closes, read after
}

func newFuture() *future {
	return &future{done: make(chan struct{})}
}

// Get implements Future.
func (f *future) Get() any {
	<-f.done
	return f.value
}

// Done implements Future.
func (f *future) Done() <-chan struct{} { return f.done }

// put publishes v. Only the first call has an effect, so a task that is
// retried and then succeeds publishes exactly one result.
func (f *future) put(v any) {
	if f.filled.CompareAndSwap(false, true) {
		f.value = v
		close(f.done)
	}
}
