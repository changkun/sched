/*
Package goscheduler implements a task scheduler with data persistence

Usage

Callers must initialize goscheduler database to use goschduler.
goschduler schedules different tasks in different goroutine and execute each task
when execution time arrival:

	// initialize the database of goscheduler
	goscheduler.Init(&goscheduler.Config{DatabaseURI: "redis://127.0.0.1:6379/8"})

	// recover tasks unfinished
	goscheduler.Poll(&task)

	// schedule a task at a specific time
	goscheduler.Schedule(&task)

	// boot a task immediately
	goscheduler.Boot(&task)

Task interface

A Task that can be scheduled by goscheduler must implements the following five methods:

	func (c YourTask)  Identifier() string
	func (c YourTask)  GetExecuteTime() time.Time
	func (c *YourTask) SetExecuteTime(t time.Time) time.Time
	func (c *YourTask) Execute()
	func (c YourTask)  FailRetryDuration() time.Duration

Note that YourTask must be a serilizable struct by `json.Marshal()`,
otherwise it cannot be scheduled by goshceudler (e.g. `type Func func()` cannot be scheduled)
*/
package goscheduler
