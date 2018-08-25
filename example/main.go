package main

import (
	"fmt"
	"math/rand"
	"os"
	"runtime/trace"
	"time"

	"github.com/changkun/goscheduler"
)

type task struct {
	Info  string
	Retry bool
	Count int

	id        string
	execution time.Time
}

func (c *task) GetID() string {
	return c.id
}
func (c *task) SetID(id string) {
	c.id = id
}

func (c *task) GetExecution() time.Time {
	return c.execution
}

func (c *task) GetTimeout() time.Duration {
	return time.Second
}

func (c *task) GetRetryDuration() time.Duration {
	return time.Second
}

func (c *task) SetExecution(t time.Time) (old time.Time) {
	old = c.execution
	c.execution = t
	return
}

const (
	retry = 3
	total = 100
)

func (c *task) Execute() (bool, error) {
	if c.Count >= retry {
		return false, nil
	}
	c.Count++
	return true, nil
}

func strictSleep(latest time.Time) {
	time.Sleep(latest.Sub(time.Now().UTC()) + time.Microsecond*100)
}

func main() {
	trace.Start(os.Stdout)
	defer trace.Stop()

	goscheduler.Init("redis://127.0.0.1:6379/1")
	start := time.Now().UTC()
	tasks := []*task{}
	for i := 0; i < total; i++ {
		e := start.Add(time.Microsecond * time.Duration(rand.Intn(1000000)))
		tasks = append(tasks, &task{
			Info:      "hello world!",
			Retry:     false,
			id:        fmt.Sprintf("task%d", i),
			execution: e, // add no more than 10s
		})

	}
	s := goscheduler.New()
	for _, task := range tasks {
		s.SetupAll(task)
	}
	strictSleep(start.Add(time.Second * 10))
}
