/*
Package goscheduler implements a consistently reliable task scheduler

Introduction

goscheduler is a consistently reliable embedded task scheduler package for GO, which applies
to be a microkernel of an internal application service, and pluggable tasks must implements
goscheduler Task interface.

goscheduler not only schedules a task at a specific time or reschedules a planned task immediately,
but also flexible to support periodically tasks, which differ from traditional non-consistently
unreliable cron task scheduling.

Furthermore, goscheduler uses priority queue schedules all tasks, and
a distributed lock mechanism that ensures tasks can only be executed once across multiple
replica instances.

Usage

Callers must initialize goscheduler database to use goschduler.
goschduler schedules different tasks in a priority queue and schedules task with minimum goroutines when tasks with same execution time arrival:

	// Init goscheduler database
	goscheduler.Init("redis://127.0.0.1:6379/1")

	// Create a temporal scheduler
	s := goscheduler.New()

	// Recover task
	s.RecoverAll(
		&ArbitraryTask1{},
		&ArbitraryTask2{},
	)

	// Setup a task
	s.Setup(&ArbitraryTask{...})

	// Launch a task
	s.Launch(&ArbitraryTask{...})

Task interface

A Task that can be scheduled by goscheduler must implements the following methods:

	// Interface for a schedulable
	type Interface interface {
		GetID() (id string)
		GetExecution() (execute time.Time)
		GetTimeout() (executeTimeout time.Duration)
		GetRetryDuration() (duration time.Duration)
		SetID(id string)
		SetExecution(new time.Time) (old time.Time)
		Execute() (retry bool, fail error)
	}

Note that your task must be a serilizable struct by `json.Marshal()`,
otherwise it cannot be persist by goshceudler (e.g. `type Func func()` cannot be scheduled)


*/
package goscheduler
