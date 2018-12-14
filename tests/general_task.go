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
	Public    string
	id        string
	execution time.Time
}

// NewTask creates a task
func NewTask(id string, e time.Time) *Task {
	return &Task{
		Public:    "not nil",
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

// GetRetryTime get retry execution duration
func (t *Task) GetRetryTime() time.Time {
	return time.Now().UTC().Add(time.Second)
}

// SetID sets the id of a task
func (t *Task) SetID(id string) {
	t.id = id
}

// IsValidID check id is valid
func (t *Task) IsValidID() bool {
	return true
}

// SetExecution sets the execution time of a task
func (t *Task) SetExecution(current time.Time) (old time.Time) {
	old = t.execution
	t.execution = current
	return
}

// Execute is the actual execution block
func (t *Task) Execute() (result interface{}, retry bool, fail error) {
	O.Push(t.id)
	return fmt.Sprintf("execute task %s.", t.id), false, nil
}
