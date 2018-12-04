// Copyright 2018 Changkun Ou. All rights reserved.
// Use of this source code is governed by a MIT
// license that can be found in the LICENSE file.

/*
Package sched provides a consistently reliable task scheduler

Introduction

sched is a consistently reliable embedded task scheduler library for GO.
It applies to be a microkernel of an internal application service, and
pluggable tasks must implements sched Task interface.

sched not only schedules a task at a specific time or reschedules a planned
task immediately, but also flexible to support periodically tasks, which
differ from traditional non-consistently unreliable cron task scheduling.

Furthermore, sched manage tasks, like goroutine runtime scheduler, uses
greedy scheduling schedules all tasks and a distributed lock mechanism that
ensures tasks can only be executed once across multiple replica instances.

Usage

Callers must initialize sched database when using sched.
sched schedules different tasks in a priority queue and schedules task with
minimum goroutines when tasks with same execution time arrival:

	// Init sched, with tasks should recovered when reboot
	sched.Init("redis://127.0.0.1:6379/1"， &ArbitraryTask1{}, &ArbitraryTask2{})

	// Setup tasks
	sched.Submit(&ArbitraryTask1{...}, &ArbitraryTask2{...})

	// Launch a task
	sched.Trigger(&ArbitraryTask1{...}, &ArbitraryTask2{...})

	// Pause scheduling
	sched.Pause()

	// Resume scheduling
	sched.Resume()

	// Stop scheduler gracefully
	sched.Stop()

Task interface

A Task that can be scheduled by sched must implements the following task interface:

	// Task interface for sched
	type Task interface {
		GetID() (id string)
		SetID(id string)
		IsValidID() bool
		GetExecution() (execute time.Time)
		SetExecution(new time.Time) (old time.Time)
		GetTimeout() (lockTimeout time.Duration)
		GetRetryTime() (execute time.Time)
		Execute() (retry bool, fail error)
	}

Note that your task must be a serilizable struct by `json.Marshal()`,
otherwise it cannot be persist by goshceudler (e.g. `type Func func()` cannot be scheduled)
*/
package sched
