// Copyright 2018 Changkun Ou. All rights reserved.
// Use of this source code is governed by a MIT
// license that can be found in the LICENSE file.

package sched_test

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/changkun/sched"
)

// EmailTask sends one email at a given time. Every field it needs after a
// restart is exported, so that sched can persist and restore it.
type EmailTask struct {
	To string    `json:"to"`
	At time.Time `json:"at"`

	id string
}

func (t *EmailTask) ID() string               { return t.id }
func (t *EmailTask) SetID(id string)          { t.id = id }
func (t *EmailTask) Execution() time.Time     { return t.At }
func (t *EmailTask) SetExecution(e time.Time) { t.At = e }
func (t *EmailTask) Timeout() time.Duration   { return time.Minute }
func (t *EmailTask) RetryTime() time.Time     { return time.Now().UTC().Add(time.Minute) }

func (t *EmailTask) Execute() (any, bool, error) {
	if err := send(t.To); err != nil {
		return nil, true, err // run again at RetryTime
	}
	return "sent to " + t.To, false, nil
}

func send(string) error { return nil }

func Example() {
	// Restore the tasks the last run left behind.
	futures, err := sched.Init("redis://127.0.0.1:6379/0", &EmailTask{})
	if err != nil {
		log.Fatal(err)
	}
	defer sched.Stop()

	for _, f := range futures {
		fmt.Println(f.Get())
	}

	// Schedule a new one and wait for its result.
	f, err := sched.Submit(&EmailTask{To: "hi@changkun.de", At: time.Now().UTC().Add(time.Hour)})
	if err != nil {
		fmt.Println("submit:", err)
		return
	}
	fmt.Println(f.Get())
}

func ExampleFuture_Done() {
	f, err := sched.Trigger(&EmailTask{To: "hi@changkun.de"})
	if err != nil {
		log.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	select {
	case <-f.Done():
		fmt.Println(f.Get())
	case <-ctx.Done():
		fmt.Println("gave up waiting:", ctx.Err())
	}
}
