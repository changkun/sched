// Copyright 2018 Changkun Ou. All rights reserved.
// Use of this source code is governed by a MIT
// license that can be found in the LICENSE file.

package simsched_test

import (
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/changkun/sched/simsched"
	"github.com/changkun/sched/tests"
)

// sleep to wait execution, a strict wait tolerance: 100 milliseconds
func strictSleep(latest time.Time) {
	time.Sleep(latest.Sub(time.Now().UTC()) + time.Millisecond*100)
}

func TestMasiveSchedule(t *testing.T) {
	tests.O.Clear()
	start := time.Now().UTC()
	expectedOrder := []string{}
	for i := 0; i < 20; i++ {
		key := fmt.Sprintf("task-%d", i)
		task := tests.NewTask(key, start.Add(time.Millisecond*10*time.Duration(i)))
		expectedOrder = append(expectedOrder, key)
		simsched.Submit(task)
	}
	strictSleep(start.Add(time.Millisecond * 10 * 25))
	if !reflect.DeepEqual(expectedOrder, tests.O.Get()) {
		t.Errorf("execution order wrong, got: %v", tests.O.Get())
	}
}

func TestNew(t *testing.T) {
	if s := simsched.New(); s == nil {
		t.Error("new scheduler fail!")
	}
}

func TestSetup(t *testing.T) {
	tests.O.Clear()
	start := time.Now().UTC()

	// save task into database
	for i := 0; i < 10; i++ {
		key := fmt.Sprintf("task-%d", i)
		task := tests.NewRetryTask(key, start.Add(time.Millisecond*10*time.Duration(i)), 2)
		simsched.Submit(task)
	}
	want := []string{
		"task-0", "task-1", "task-2", "task-3", "task-4",
		"task-0", "task-5", "task-1", "task-6", "task-2",
		"task-7", "task-3", "task-8", "task-4", "task-0",
		"task-9", "task-5", "task-1", "task-6", "task-2",
		"task-7", "task-3", "task-8", "task-4", "task-9",
		"task-5", "task-6", "task-7", "task-8", "task-9",
	}
	strictSleep(start.Add(time.Second))
	if !reflect.DeepEqual(tests.O.Get(), want) {
		t.Errorf("setup retry task execution order is not as expected, got: %v", tests.O.Get())
	}
}

func TestSchedule1(t *testing.T) {
	tests.O.Clear()
	start := time.Now().UTC()

	task1 := tests.NewTask("task-1", start.Add(time.Millisecond*100))
	simsched.Submit(task1)
	task2 := tests.NewTask("task-2", start.Add(time.Millisecond*30))
	taskAreplica := tests.NewTask("task-1", start.Add(time.Millisecond*10))
	go simsched.Submit(task2)
	go simsched.Boot(taskAreplica)

	strictSleep(start.Add(time.Second))
	want := []string{"task-1", "task-2"}
	if !reflect.DeepEqual(tests.O.Get(), want) {
		t.Errorf("launch task execution order is not as expected, got: %v", tests.O.Get())
	}
}
func TestSchedule2(t *testing.T) {
	tests.O.Clear()
	start := time.Now().UTC()

	task1 := tests.NewTask("task-1", start.Add(time.Millisecond*100))
	simsched.Submit(task1)
	task2 := tests.NewTask("task-2", start.Add(time.Millisecond*30))
	go simsched.Submit(task2)
	go simsched.Boot(task1)

	strictSleep(start.Add(time.Second))
	want := []string{"task-1", "task-2"}
	if !reflect.DeepEqual(tests.O.Get(), want) {
		t.Errorf("launch task execution order is not as expected, got: %v", tests.O.Get())
	}
}
