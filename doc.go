// Copyright 2018 Changkun Ou. All rights reserved.
// Use of this source code is governed by a MIT
// license that can be found in the LICENSE file.

/*
Package sched schedules tasks at a given time, keeps them across restarts,
and runs each one exactly once across every replica of an application.

# Introduction

sched is an embedded scheduler. It is a library, not a service: it lives
inside the application it schedules for. A task is a Go type that implements
[Task], so it carries its own data and its own code, and sched persists it to
Redis. After a crash or a deployment the tasks that had not run yet come back
with the application.

Where cron fires a command and forgets it, sched gives every task a [Future]
that carries the result, reschedules a task that asks for a retry, and takes a
distributed lock before it runs, so that a task scheduled on five replicas
still runs once.

# Usage

Init connects to Redis and restores the tasks a previous run left behind.
Pass an empty value of every task type that has to be recoverable, because
JSON alone cannot say which type a record belongs to.

	futures, err := sched.Init("redis://127.0.0.1:6379/0",
		&EmailTask{},
		&ReportTask{},
	)
	if err != nil {
		return err
	}
	for _, f := range futures {
		fmt.Println(f.Get())
	}

Submit schedules a task for the time it reports, and Trigger schedules it for
now:

	f, err := sched.Submit(&EmailTask{To: "hi@changkun.de", At: tomorrow})
	if err != nil {
		return err
	}
	fmt.Println(f.Get())

Get blocks until the task finishes. Done gives the same result without
blocking, so a caller can give up:

	select {
	case <-f.Done():
		fmt.Println(f.Get())
	case <-ctx.Done():
		return ctx.Err()
	}

Submitting an identifier that is already scheduled moves the task to the new
time instead of adding a second one, and resolves every Future that waits for
it. Pause stops the scheduler from starting tasks, Resume starts it again,
Wait blocks until nothing is queued or running, and Stop shuts the scheduler
down after the tasks that already started have finished.

# Writing a task

A task carries its own state, so every field the task needs after a restart
has to be exported and encodable by [encoding/json]. sched restores the
identifier and the execution time itself.

	type EmailTask struct {
		To string    `json:"to"`
		At time.Time `json:"at"`

		id string
	}

	func (t *EmailTask) ID() string           { return t.id }
	func (t *EmailTask) SetID(id string)      { t.id = id }
	func (t *EmailTask) Execution() time.Time { return t.At }
	func (t *EmailTask) SetExecution(e time.Time) { t.At = e }
	func (t *EmailTask) Timeout() time.Duration  { return time.Minute }
	func (t *EmailTask) RetryTime() time.Time {
		return time.Now().UTC().Add(time.Minute)
	}

	func (t *EmailTask) Execute() (any, bool, error) {
		if err := send(t.To); err != nil {
			return nil, true, err // retry at RetryTime
		}
		return "sent", false, nil
	}

Timeout is the lifetime of the lock that keeps two replicas from running the
same task. Keep it shorter than the time Execute needs, or a second replica
can start while the first one still runs.

# What a Future returns

Get returns whatever Execute returned, or an error. A task that panics
produces an error that names the task and the panic. A task that sched
refused to run produces one of [ErrTaskClaimed], [ErrTaskUnverifiable] or
[ErrStopped], so a caller is never left waiting for a result that cannot
arrive.

# How it works

One goroutine owns a priority queue of tasks ordered by execution time, and
one timer serves the task at the head of that queue. Callers never touch the
queue. Submit persists the task, appends it to a wait-free multi-producer
queue and signals the scheduler, all in a bounded number of atomic steps, so
no caller ever waits for a lock to schedule a task. The package holds no
mutex.

Each task that comes due runs in its own goroutine, so a slow task delays
neither the scheduler nor the tasks behind it. Before it runs, the goroutine
takes the Redis lock of the task and rereads the persisted execution time: if
another replica moved the task, this one reschedules instead of running early.
*/
package sched
