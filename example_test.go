package goscheduler_test

import (
	"fmt"
	"time"

	"github.com/changkun/goscheduler"
	"github.com/changkun/goscheduler/internal/store"
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

	goscheduler.Init("redis://127.0.0.1:6379/1")
	// Output:
	//

}

func ExampleScheduler_Recover() {

	goscheduler.Init("redis://127.0.0.1:6379/1")
	execution := time.Now().UTC().Add(time.Second)
	task := &ExampleTask{
		Info:      "hello world!",
		id:        "task-unique-id",
		execution: execution,
	}
	store.Save(task)

	goscheduler.New().Recover(&ExampleTask{})
	strictSleep(execution.Add(time.Second))

	// Output:
	// ExampleTask task-unique-id is scheduled: hello world!
}

func ExampleScheduler_Setup() {

	goscheduler.Init("redis://127.0.0.1:6379/1")

	execution := time.Now().UTC().Add(time.Second)
	task := &ExampleTask{
		Info:      "hello world!",
		id:        "task-unique-id",
		execution: execution,
	}
	if err := goscheduler.New().Setup(task); err != nil {
		fmt.Println("Schedule task failed: ", err)
	}
	strictSleep(execution.Add(time.Second))

	// Output:
	// ExampleTask task-unique-id is scheduled: hello world!

}

func ExampleScheduler_Launch() {

	goscheduler.Init("redis://127.0.0.1:6379/1")

	execution := time.Now().UTC().Add(time.Second * 10)
	task := &ExampleTask{
		Info:      "hello world!",
		id:        "task-unique-id",
		execution: execution,
	}
	if err := goscheduler.New().Setup(task); err != nil {
		fmt.Println("Schedule task failed: ", err)
	}

	if err := goscheduler.New().Launch(task); err != nil {
		fmt.Println("Schedule task failed: ", err)
	}
	strictSleep(time.Now().UTC().Add(time.Second))

	// Output:
	// ExampleTask task-unique-id is scheduled: hello world!
}
