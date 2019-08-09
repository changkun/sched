package main

import (
	"fmt"
	"time"

	"github.com/changkun/sched"
	"github.com/changkun/sched/tests"
)

func main() {
	sched.Init("redis://127.0.0.1:6379/2")
	defer sched.Stop()

	start := time.Now().UTC()
	expectedOrder := []string{}

	futures := make([]sched.TaskFuture, 20)
	for i := 0; i < 20; i++ {
		key := fmt.Sprintf("task-%d", i)
		task := tests.NewTask(key, start.Add(time.Millisecond*10*time.Duration(i)))
		expectedOrder = append(expectedOrder, key)
		future, _ := sched.Submit(task)
		futures[i] = future
	}
	for i := range futures {
		fmt.Println(futures[i].Get())
	}
}
