// Copyright 2018 Changkun Ou. All rights reserved.
// Use of this source code is governed by a MIT
// license that can be found in the LICENSE file.

package tests

import (
	"fmt"
	"time"
)

// Task implements task.Interface
type Task struct {
	id        string
	execution time.Time
}

// NewTask creates a task
func NewTask(id string, e time.Time) *Task {
	return &Task{
		id:        id,
		execution: e,
	}
}

// GetID get task id
func (t *Task) GetID() (id string) {
	id = t.id
	return
}

// GetExecution get execution time
func (t *Task) GetExecution() (execute time.Time) {
	execute = t.execution
	return
}

// GetTimeout get timeout of execution
func (t *Task) GetTimeout() (executeTimeout time.Duration) {
	return time.Second
}

// GetRetryDuration get retry execution duration
func (t *Task) GetRetryDuration() (duration time.Duration) {
	return time.Second
}

// SetID sets the id of a task
func (t *Task) SetID(id string) {
	t.id = id
}

// SetExecution sets the execution time of a task
func (t *Task) SetExecution(current time.Time) (old time.Time) {
	old = t.execution
	t.execution = current
	return
}

// Execute is the actual execution block
func (t *Task) Execute() (retry bool, fail error) {
	O.Push(t.id)
	fmt.Printf("execute task %s.\n", t.id)
	return false, nil
}
