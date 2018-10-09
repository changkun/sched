package main_test

import (
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/changkun/sched/simsched"
	"github.com/changkun/sched/tests"
)

func strictSleep(latest time.Time) {
	time.Sleep(latest.Sub(time.Now().UTC()) + time.Millisecond*100)
}

func TestSched(t *testing.T) {
	const (
		retry = 3
		total = 1000
	)

	start := time.Now().UTC()
	tasks := []*tests.SimRetryTask{}
	for i := 0; i < total; i++ {
		e := start.Add(time.Millisecond * time.Duration(rand.Intn(10000)))
		tasks = append(tasks, tests.NewSimRetryTask(fmt.Sprintf("task-%d", i), e, retry))
	}
	for _, t := range tasks {
		simsched.Submit(t)
	}
	strictSleep(start.Add(time.Second * 20))
}
