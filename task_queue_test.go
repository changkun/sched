// Copyright 2018 Changkun Ou. All rights reserved.
// Use of this source code is governed by a MIT
// license that can be found in the LICENSE file.

package sched

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/changkun/sched/tests"
)

func TestTaskQueue(t *testing.T) {
	// Some items and their priorities.
	start := time.Now().UTC()
	tpq := newTaskQueue()

	// Insert a new item and then modify its priority.
	task := tests.NewTask("task-0", start.Add(time.Millisecond*1))
	tpq.Push(task)

	// Insert a new item and then modify its priority.
	wg := sync.WaitGroup{}
	for i := 1; i < 10; i++ {
		wg.Add(1)
		go func(i int) {
			// the task should ordered by task-number
			task := tests.NewTask(fmt.Sprintf("task-%d", i), start.Add(time.Millisecond*5*time.Duration(i)))
			tpq.Push(task)
			wg.Done()
		}(i)
	}
	wg.Wait()

	p := tpq.Peek()
	if p.GetID() != "task-0" {
		t.Errorf("first task must have id of task-0")
	}

	for i := 0; i < 10; i++ {
		task := tests.NewTask(fmt.Sprintf("task-%d", i), start.Add(-time.Millisecond*5*time.Duration(i)))
		tpq.Update(task)
	}

	l := tpq.Len()
	for i := 0; i < l; i++ {
		task := tpq.Pop()
		want := fmt.Sprintf("task-%d", l-1-i)
		if task.Value.GetID() != want {
			t.Errorf("task has improper task id, want %s, got %s", want, task.Value.GetID())
		}
	}
}

func TestTaskQueue_PushFail(t *testing.T) {
	// Some items and their priorities.
	start := time.Now().UTC()
	tpq := newTaskQueue()

	// Insert a new item and then modify its priority.
	task := tests.NewTask("task-0", start.Add(time.Millisecond*1))
	if _, ok := tpq.Push(task); !ok {
		t.Error("first push must success!")
	}
	if _, ok := tpq.Push(task); ok {
		t.Error("second push must fail!")
	}
}

func TestTaskQueue_PopFail(t *testing.T) {
	tpq := newTaskQueue()
	if tt := tpq.Pop(); tt != nil {
		t.Error("pop from empty queue must return nil!")
	}
}

func TestTaskQueue_PeekFail(t *testing.T) {
	tpq := newTaskQueue()
	tt := tpq.Peek()
	if tt != nil {
		t.Errorf("peek an empty task queue must be empty, got: %v", tt)
	}
}

func TestTaskQueue_UpdateFail(t *testing.T) {
	tpq := newTaskQueue()
	task := tests.NewTask("task-0", time.Now().UTC().Add(time.Millisecond*1))
	if _, ok := tpq.Update(task); ok {
		t.Errorf("update non existing task must be fail!")
	}
}
