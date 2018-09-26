// Copyright 2018 Changkun Ou. All rights reserved.
// Use of this source code is governed by a MIT
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"math/rand"
	"os"
	"runtime/trace"
	"time"

	"github.com/changkun/sched"
	"github.com/changkun/sched/tests"
)

func strictSleep(latest time.Time) {
	time.Sleep(latest.Sub(time.Now().UTC()) + time.Millisecond*100)
}

func init() {
	sched.Init("redis://127.0.0.1:6379/1")
}

func main() {
	f, _ := os.Create("bench.trace")
	defer f.Close()

	if err := trace.Start(f); err != nil {
		panic(err)
	}
	defer trace.Stop()

	tests.O.Clear()
	const (
		retry = 3
		total = 1000
	)
	start := time.Now().UTC()
	min := start.Add(time.Millisecond * 10000)
	max := start
	tasks := []*tests.RetryTask{}
	for i := 0; i < total; i++ {
		e := start.Add(time.Millisecond * time.Duration(rand.Intn(10000)))
		if e.After(max) {
			max = e
		}
		if e.Before(min) {
			min = e
		}
		tasks = append(tasks, tests.NewRetryTask(fmt.Sprintf("task-%d", i), e, retry))
	}
	for _, t := range tasks {
		if err := sched.Setup(t); err != nil {
			fmt.Printf("setup task %s error: %v\n", t.GetID(), err)
		}
	}
	strictSleep(start.Add(time.Second * 20))

	count := len(tests.O.Get())
	fmt.Printf("                   %d Execution in %s   \n", count, max.Sub(min))
	fmt.Println("--------------------------------------------------------------")
	fmt.Println("        required execution: ", retry*total)
	fmt.Println("          actual execution: ", count)
	fmt.Println("--------------------------------------------------------------")
	fmt.Println("   first required schedule: ", min.Format(time.StampNano))
	fmt.Println("     first actual schedule: ", tests.O.GetFirst().Format(time.StampNano))
	fmt.Println("--------------------------------------------------------------")
	fmt.Println("    last required schedule: ", max.Format(time.StampNano))
	fmt.Println("      last actual schedule: ", tests.O.GetLast().Format(time.StampNano))
	fmt.Println("--------------------------------------------------------------")
	fmt.Println("      first schedule delay: ", tests.O.GetFirst().Sub(min))
	fmt.Println("       last schedule delay: ", tests.O.GetLast().Sub(max))
	fmt.Println("--------------------------------------------------------------")
	fmt.Println("required execution density: ", time.Duration(int(max.Sub(min))/count))
	fmt.Println("  actual execution density: ", time.Duration(int(tests.O.GetLast().Sub(tests.O.GetFirst()))/count))
	fmt.Println("--------------------------------------------------------------")
}
