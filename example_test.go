// Copyright 2018 Changkun Ou. All rights reserved.
// Use of this source code is governed by a MIT
// license that can be found in the LICENSE file.

package sched_test

import (
	"fmt"
	"time"

	"github.com/changkun/sched"
	"github.com/changkun/sched/internal/store"
	"github.com/changkun/sched/tests"
)

func ExampleInit() {

	sched.Init("redis://127.0.0.1:6379/1")
	// Output:
	//

}

func ExampleScheduler_Recover() {

	sched.Init("redis://127.0.0.1:6379/1")
	execution := time.Now().UTC().Add(time.Second)
	task := tests.NewTask("task-unique-id", execution)
	if err := store.Save(task); err != nil {
		fmt.Printf("store with task-unique-id error: %v\n", err)
		return
	}
	if err := sched.Recover(&tests.Task{}); err != nil {
		fmt.Printf("recover task with task-unique-id error: %v\n", err)
		return
	}
	strictSleep(execution.Add(time.Second))

	// Output:
	// Execute task task-unique-id.
}

func ExampleScheduler_Setup() {

	sched.Init("redis://127.0.0.1:6379/1")

	execution := time.Now().UTC().Add(time.Second)
	task := tests.NewTask("task-unique-id", execution)
	if err := sched.Setup(task); err != nil {
		fmt.Printf("Schedule task failed: %v\n", err)
	}
	strictSleep(execution.Add(time.Second))

	// Output:
	// Execute task task-unique-id.

}

func ExampleScheduler_Launch() {

	sched.Init("redis://127.0.0.1:6379/1")

	execution := time.Now().UTC().Add(time.Second * 10)
	task := tests.NewTask("task-unique-id", execution)
	if err := sched.Setup(task); err != nil {
		fmt.Printf("Schedule task failed: %v\n", err)
	}
	if err := sched.Launch(task); err != nil {
		fmt.Printf("Schedule task failed: %v\n", err)
	}
	strictSleep(time.Now().UTC().Add(time.Second))

	// Output:
	// Execute task task-unique-id.
}
