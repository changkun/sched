// Copyright 2018 Changkun Ou. All rights reserved.
// Use of this source code is governed by a MIT
// license that can be found in the LICENSE file.

package tests

import (
	"fmt"
	"sync"
	"time"
)

// RetryTask implements task.Interface However it has no
// export field, which is not able to be scheduled by sched
type RetryTask struct {
	RetryCount int
	MaxRetry   int
	id         string
	execution  time.Time
	mu         sync.Mutex
}

// NewRetryTask creates a task
func NewRetryTask(id string, e time.Time, maxRetry int) *RetryTask {
	return &RetryTask{
		RetryCount: 0,
		MaxRetry:   maxRetry,
		id:         id,
		execution:  e,
	}
}

// GetID get task id
func (t *RetryTask) GetID() (id string) {
	id = t.id
	return
}

// GetExecution get execution time
func (t *RetryTask) GetExecution() (execute time.Time) {
	t.mu.Lock()
	defer t.mu.Unlock()
	execute = t.execution
	return
}

// GetTimeout get timeout of execution
func (t *RetryTask) GetTimeout() (executeTimeout time.Duration) {
	return time.Second
}

// GetRetryDuration get retry execution duration
func (t *RetryTask) GetRetryDuration() (duration time.Duration) {
	return time.Millisecond * 42
}

// SetID sets the id of a task
func (t *RetryTask) SetID(id string) {
	t.id = id
}

// SetExecution sets the execution time of a task
func (t *RetryTask) SetExecution(current time.Time) (old time.Time) {
	t.mu.Lock()
	defer t.mu.Unlock()
	old = t.execution
	t.execution = current
	return
}

// Execute is the actual execution block
func (t *RetryTask) Execute() (retry bool, fail error) {
	if t.RetryCount > t.MaxRetry {
		O.SetLast(time.Now().UTC())
		fmt.Printf("Execute retry task %s, retry count: %d, tollerance: %v, last retry.\n", t.id, t.RetryCount, time.Now().UTC().Sub(t.GetExecution()))
		return false, nil

	}
	O.Push(t.id)
	if O.IsFirstZero() {
		O.SetFirst(time.Now().UTC())
	}
	fmt.Printf("Execute retry task %s, retry count: %d. tollerance: %v\n", t.id, t.RetryCount, time.Now().UTC().Sub(t.GetExecution()))
	t.RetryCount++
	return true, nil
}
