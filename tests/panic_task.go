// Copyright 2018 Changkun Ou. All rights reserved.
// Use of this source code is governed by a MIT
// license that can be found in the LICENSE file.

package tests

import (
	"time"
)

// PanicTask implements task.Interface
type PanicTask struct {
	Public    string
	id        string
	execution time.Time
}

// NewPanicTask creates a task
func NewPanicTask(id string, e time.Time) *PanicTask {
	return &PanicTask{
		Public:    "not nil",
		id:        id,
		execution: e,
	}
}

// GetID get task id
func (t *PanicTask) GetID() (id string) {
	id = t.id
	return
}

// GetExecution get execution time
func (t *PanicTask) GetExecution() (execute time.Time) {
	execute = t.execution
	return
}

// GetTimeout get timeout of execution
func (t *PanicTask) GetTimeout() (executeTimeout time.Duration) {
	return time.Second
}

// GetRetryTime get retry execution duration
func (t *PanicTask) GetRetryTime() time.Time {
	return time.Now().UTC().Add(time.Second)
}

// SetID sets the id of a task
func (t *PanicTask) SetID(id string) {
	t.id = id
}

// IsValidID check id is valid
func (t *PanicTask) IsValidID() bool {
	return true
}

// SetExecution sets the execution time of a task
func (t *PanicTask) SetExecution(current time.Time) (old time.Time) {
	old = t.execution
	t.execution = current
	return
}

// Execute is the actual execution block
func (t *PanicTask) Execute() (result interface{}, retry bool, fail error) {
	O.Push(t.id)
	panic(t.id)
}
