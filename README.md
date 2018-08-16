# goscheduler

[![GoDoc](https://godoc.org/github.com/changkun/goscheduler?status.svg)](https://godoc.org/github.com/changkun/goscheduler) [![Build Status](https://travis-ci.org/changkun/goscheduler.svg?branch=master)](https://travis-ci.org/changkun/goscheduler) [![Go Report Card](https://goreportcard.com/badge/github.com/changkun/goscheduler)](https://goreportcard.com/report/github.com/changkun/goscheduler) [![codecov](https://codecov.io/gh/changkun/goscheduler/branch/master/graph/badge.svg)](https://codecov.io/gh/changkun/goscheduler) ![](https://img.shields.io/github/release/changkun/goscheduler/all.svg)

goschduler is consistently reliable task scheduler library for _GO_.

## Features

- Thread safety
- Distributed consistency
- Schedule a task at a specific time
- Boot (an existing) task immediately
- Recover specified type of tasks from database when app restarts
- Retry if scheduled task faild

## Usage

goscheduler uses [Redis](https://redis.io/) for data persistence, 
it schedules your task at a specific time or boot an existing task immediately.

```go
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

// Identifier must returns a unique string for the task, usually can be an UUID
func (c CustomTask) Identifier() string {
	return c.ID
}

// GetExecuteTime must returns the excute time of the task
func (c CustomTask) GetExecuteTime() time.Time {
	return c.End
}

// SetExecuteTime can set the execution time of the task
// It allows goscheduler set a retry execution time for your task
func (c *CustomTask) SetExecuteTime(t time.Time) time.Time {
	c.End = t
	return c.End
}

// Execute defines the actual running task,
// returning an error means the execution is failed
func (c *CustomTask) Execute() error {
	// implement your task execution
	fmt.Println("Task is Running: ", c.Information)
	return nil
}

// FailRetryDuration returns the task retry duration if your task failed
func (c CustomTask) FailRetryDuration() time.Duration {
	return time.Second
}

func main() {
	// Init goscheduler database
	goscheduler.Init(&goscheduler.Config{
		DatabaseURI: "redis://127.0.0.1:6379/8",
	})

	// When goscheduler database is initiated,
	// call Poller to recover all unfinished task
	var task CustomTask
    goscheduler.Poll(&task)
    // the task variable is only used for determing your task type
    // the task remains nil after Poll()

	// A task should be executed in 10 seconds
	task = CustomTask{
		ID:          "123",
		Start:       time.Now().UTC(),
		End:         time.Now().UTC().Add(time.Duration(10) * time.Second),
		Information: "this is a task message message",
	}
	fmt.Println("Retry duration if execution failed: ", task.FailRetryDuration())

	// first schedule the task at 10 seconds later
	goscheduler.Schedule(&task)
	// however we decide to boot the task immediately
	goscheduler.Boot(&task)

	// let's sleep 2 secs wait for the retult of the task
	time.Sleep(time.Second * 2)
}
```

## License

[MIT](./LICENSE)