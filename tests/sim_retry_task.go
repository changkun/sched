// Copyright 2018 Changkun Ou. All rights reserved.
// Use of this source code is governed by a MIT
// license that can be found in the LICENSE file.

package tests

import (
	"time"
)

// SimRetryTask implements task.Interface However it has no
// export field, which is not able to be scheduled by sched
type SimRetryTask struct {
	RetryCount int
	MaxRetry   int
	id         string
	execution  time.Time
}

// NewSimRetryTask creates a task
func NewSimRetryTask(id string, e time.Time, maxRetry int) *SimRetryTask {
	return &SimRetryTask{
		MaxRetry:  maxRetry,
		id:        id,
		execution: e,
	}
}

// GetID get task id
func (t *SimRetryTask) GetID() (id string) {
	id = t.id
	return
}

// GetExecution get execution time
func (t *SimRetryTask) GetExecution() (execute time.Time) {
	execute = t.execution
	return
}

// GetTimeout get timeout of execution
func (t *SimRetryTask) GetTimeout() (executeTimeout time.Duration) {
	return time.Second
}

// GetRetryDuration get retry execution duration
func (t *SimRetryTask) GetRetryDuration() (duration time.Duration) {
	return time.Millisecond * 42
}

// SetID sets the id of a task
func (t *SimRetryTask) SetID(id string) {
	t.id = id
}

// SetExecution sets the execution time of a task
func (t *SimRetryTask) SetExecution(current time.Time) (old time.Time) {
	old = t.execution
	t.execution = current
	return
}

// Execute is the actual execution block
func (t *SimRetryTask) Execute() (retry bool, fail error) {
	if t.RetryCount > t.MaxRetry {
		return false, nil
	}
	t.RetryCount++
	return true, nil
}
