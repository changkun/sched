// Copyright 2018 Changkun Ou. All rights reserved.
// Use of this source code is governed by a MIT
// license that can be found in the LICENSE file.

package sched

import (
	"time"
)

// Task interface for sched
type Task interface {
	// GetID must returns a unique ID for all of the scheduled task.
	GetID() (id string)
	// SetID will set id as the unique ID for the scheduled task.
	SetID(id string)
	// IsValidID verifies that an ID is an valid ID for the task.
	IsValidID() bool
	// GetExecution returns the time for task execution.
	GetExecution() (execute time.Time)
	// SetExecution sets a new time for the task execution
	SetExecution(new time.Time) (old time.Time)
	// GetTimeout returns the locking time for a giving task.
	// Users should aware that this time should *not* longer than the task execution time.
	// For instance, if your task consumes 1 second for execution,
	// then the locking time must shorter than 1 second.
	GetTimeout() (lockTimeout time.Duration)
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
