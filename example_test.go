package sched_test

import (
	"fmt"
	"time"

	"github.com/changkun/sched"
	"github.com/changkun/sched/internal/store"
)

type ExampleTask struct {
	Info      string
	id        string
	execution time.Time
}

// ID implements a task can be scheduled by scheduler
func (c *ExampleTask) GetID() string {
	return c.id
}
func (c *ExampleTask) SetID(id string) {
	c.id = id
}

func (c *ExampleTask) GetExecution() time.Time {
	return c.execution
}

func (c *ExampleTask) GetTimeout() time.Duration {
	return time.Second
}

func (c *ExampleTask) GetRetryDuration() time.Duration {
	return time.Second
}

func (c *ExampleTask) SetExecution(t time.Time) (old time.Time) {
	old = c.execution
	c.execution = t
	return
}

func (c *ExampleTask) Execute() (bool, error) {
	fmt.Printf("ExampleTask %s is scheduled: %s\n", c.id, c.Info)
	return false, nil
}

func ExampleInit() {

	sched.Init("redis://127.0.0.1:6379/1")
	// Output:
	//

}

func ExampleScheduler_Recover() {

	sched.Init("redis://127.0.0.1:6379/1")
	execution := time.Now().UTC().Add(time.Second)
	task := &ExampleTask{
		Info:      "hello world!",
		id:        "task-unique-id",
		execution: execution,
	}
	if err := store.Save(task); err != nil {
		fmt.Printf("store task-unique-id error: %v\n", err)
		return
	}

	if err := sched.Recover(&ExampleTask{}); err != nil {
		fmt.Printf("recover task-unique-id error: %v\n", err)
		return
	}
	strictSleep(execution.Add(time.Second))

	// Output:
	// ExampleTask task-unique-id is scheduled: hello world!
}

func ExampleScheduler_Setup() {

	sched.Init("redis://127.0.0.1:6379/1")

	execution := time.Now().UTC().Add(time.Second)
	task := &ExampleTask{
		Info:      "hello world!",
		id:        "task-unique-id",
		execution: execution,
	}
	if err := sched.Setup(task); err != nil {
		fmt.Printf("Schedule task failed: %v\n", err)
	}
	strictSleep(execution.Add(time.Second))

	// Output:
	// ExampleTask task-unique-id is scheduled: hello world!

}

func ExampleScheduler_Launch() {

	sched.Init("redis://127.0.0.1:6379/1")

	execution := time.Now().UTC().Add(time.Second * 10)
	task := &ExampleTask{
		Info:      "hello world!",
		id:        "task-unique-id",
		execution: execution,
	}
	if err := sched.Setup(task); err != nil {
		fmt.Printf("Schedule task failed: %v\n", err)
	}

	if err := sched.Launch(task); err != nil {
		fmt.Printf("Schedule task failed: %v\n", err)
	}
	strictSleep(time.Now().UTC().Add(time.Second))

	// Output:
	// ExampleTask task-unique-id is scheduled: hello world!
}
