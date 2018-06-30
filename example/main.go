package main

import (
	"fmt"
	"time"

	"github.com/changkun/goscheduler"
)

// CustomTask define your custom task struct
type CustomTask struct {
	ID          string    `json:"uuid"`
	Start       time.Time `json:"start"`
	End         time.Time `json:"end"`
	Information string    `json:"info"`
}

// Identifier implement the following methods
func (c CustomTask) Identifier() string {
	return c.ID
}

// GetExecuteTime returns the executing time of the task
func (c CustomTask) GetExecuteTime() time.Time {
	return c.End
}

// SetExecuteTime sets the executing time of the task
func (c *CustomTask) SetExecuteTime(t time.Time) time.Time {
	c.End = t
	return c.End
}

// Execute the task
func (c CustomTask) Execute() {
	// implement your task execution in
	fmt.Println("Task is Running: ", c.Information)
}

func main() {
	// Init scheduler database
	goscheduler.Initialize(&goscheduler.DatabaseConfig{
		URI: "redis://127.0.0.1:6379/8",
	})

	// A task should be execute in 10 seconds
	task := &CustomTask{
		ID:          "123",
		Start:       time.Now().UTC(),
		End:         time.Now().UTC().Add(time.Duration(10) * time.Second),
		Information: "this is a task message message",
	}
	// first schedule the task at 10 seconds later
	goscheduler.Schedule(task)
	// however we decide to boot the task immediately
	goscheduler.Boot(task)

	time.Sleep(time.Second * 2)
}
