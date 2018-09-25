package main

import (
	"fmt"
	"math/rand"
	"os"
	"runtime/trace"
	"sync"
	"time"

	"github.com/changkun/sched"
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

var (
	first   time.Time
	last    time.Time
	counter = 0
	mlock   sync.Mutex
)

func (c *task) Execute() (bool, error) {
	if c.Count >= retry {
		return false, nil
	}

	// for benckmarks
	mlock.Lock()
	if first.IsZero() {
		first = time.Now().UTC()
	}
	last = time.Now().UTC()
	counter++
	mlock.Unlock()

	fmt.Printf("task %s is scheduled: %s, retry: %d, execution tolerance: %s\n", c.id, c.Info, c.Count, time.Now().UTC().Sub(c.GetExecution()))
	c.Count++
	return true, nil
}

func strictSleep(latest time.Time) {
	time.Sleep(latest.Sub(time.Now().UTC()) + time.Millisecond*100)
}

func main() {
	f, _ := os.Create("bench.trace")
	defer f.Close()

	if err := trace.Start(f); err != nil {
		panic(err)
	}
	defer trace.Stop()

	sched.Init("redis://127.0.0.1:6379/1")

	start := time.Now().UTC()
	min := start.Add(time.Millisecond * 10000)
	max := start
	tasks := []*task{}
	for i := 0; i < total; i++ {
		e := start.Add(time.Millisecond * time.Duration(rand.Intn(10000)))
		if e.After(max) {
			max = e
		}
		if e.Before(min) {
			min = e
		}
		tasks = append(tasks, &task{
			Info:      "hello world!",
			Retry:     false,
			id:        fmt.Sprintf("task%d", i),
			execution: e,
		})
	}
	s := sched.New()
	for _, t := range tasks {
		go func(t *task) {
			if err := s.Setup(t); err != nil {
				fmt.Printf("setup task %s error: %s\n", t.GetID(), err.Error())
			}
		}(t)
	}
	strictSleep(start.Add(time.Second * 13))

	mlock.Lock()
	fmt.Printf("                   %d Execution in %s   \n", counter, max.Sub(min))
	fmt.Println("--------------------------------------------------------------")
	fmt.Println("        required execution: ", retry*total)
	fmt.Println("          actual execution: ", counter)
	fmt.Println("--------------------------------------------------------------")
	fmt.Println("   first required schedule: ", min.Format(time.StampNano))
	fmt.Println("     first actual schedule: ", first.Format(time.StampNano))
	fmt.Println("--------------------------------------------------------------")
	fmt.Println("    last required schedule: ", max.Format(time.StampNano))
	fmt.Println("      last actual schedule: ", last.Format(time.StampNano))
	fmt.Println("--------------------------------------------------------------")
	fmt.Println("      first schedule delay: ", first.Sub(min))
	fmt.Println("       last schedule delay: ", last.Sub(max))
	fmt.Println("--------------------------------------------------------------")
	fmt.Println("required execution density: ", time.Duration(int(max.Sub(min))/counter))
	fmt.Println("  actual execution density: ", time.Duration(int(last.Sub(first))/counter))
	fmt.Println("--------------------------------------------------------------")
	mlock.Unlock()
}
