package goscheduler_test

import (
	"fmt"
	"math/rand"
	"sync"
	"testing"
	"time"

	"github.com/changkun/goscheduler"
	"github.com/changkun/goscheduler/internal/store"
	"github.com/stretchr/testify/assert"
)

// ArbitraryTask implements task.Interface that can be scheduled by schedule
type ArbitraryTask struct {
	Info  string
	Retry bool
	Count int

	id        string
	execution time.Time
}

// ID implements a task can be scheduled by scheduler
func (c *ArbitraryTask) GetID() string {
	return c.id
}
func (c *ArbitraryTask) SetID(id string) {
	c.id = id
}

func (c *ArbitraryTask) GetExecution() time.Time {
	return c.execution
}

func (c *ArbitraryTask) GetTimeout() time.Duration {
	return time.Second
}

func (c *ArbitraryTask) GetRetryDuration() time.Duration {
	return time.Second
}

func (c *ArbitraryTask) SetExecution(t time.Time) (old time.Time) {
	old = c.execution
	c.execution = t
	return
}

const (
	Total = 100
	Retry = 3
)

var (
	last    time.Time
	counter = 0
	mlock   sync.Mutex
)

func (c *ArbitraryTask) Execute() (bool, error) {
	if c.Count >= Retry {
		return false, nil
	}

	mlock.Lock()
	last = time.Now().UTC()
	counter++
	mlock.Unlock()

	fmt.Printf("ArbitraryTask %s is scheduled: %s, count: %d, execution tolerance: %s\n", c.id, c.Info, c.Count, time.Now().UTC().Sub(c.GetExecution()))
	c.Count++
	return true, nil
}

// sleep to wait execution, a strict wait tolerance: 100 milliseconds
func strictSleep(latest time.Time) {
	time.Sleep(latest.Sub(time.Now().UTC()) + time.Millisecond*100)
}

func init() {
	goscheduler.Init("redis://127.0.0.1:6379/1")
}
func TestScheduleSetup(t *testing.T) {
	// clear counter
	mlock.Lock()
	counter = 0
	mlock.Unlock()

	start := time.Now().UTC()
	max := start
	tasks := []*ArbitraryTask{}
	for i := 0; i < Total; i++ {
		e := start.Add(time.Millisecond * time.Duration(rand.Intn(1000)))
		if e.After(max) {
			max = e
		}
		tasks = append(tasks, &ArbitraryTask{
			Info:      "hello world!",
			Retry:     false,
			id:        fmt.Sprintf("task%d", i),
			execution: e, // add no more than 10s
		})

	}
	s := goscheduler.New()
	for _, task := range tasks {
		assert.Nil(t, s.SetupAll(task))
	}

	strictSleep(max.Add(time.Second * 3))
	mlock.Lock()
	fmt.Println("total delay: ", last.Sub(max))
	fmt.Println("last required schedule: ", max)
	fmt.Println("actual schedule:        ", last)
	assert.Equal(t, Retry*Total, counter, "tasks must be scheduled without missing!")
	mlock.Unlock()
}

func TestScheduleLaunch(t *testing.T) {
	// clear counter
	mlock.Lock()
	counter = 0
	mlock.Unlock()

	start := time.Now().UTC()
	max := start
	tasks := []*ArbitraryTask{}
	for i := 0; i < Total; i++ {
		e := start.Add(time.Millisecond * time.Duration(rand.Intn(1000)))
		if e.After(max) {
			max = e
		}
		tasks = append(tasks, &ArbitraryTask{
			Info:      "hello world!",
			Retry:     false,
			id:        fmt.Sprintf("task%d", i),
			execution: e,
		})
	}

	s := goscheduler.New()

	// Schedule all first
	for _, task := range tasks {
		assert.Nil(t, s.SetupAll(task))
	}

	// Launch all by then
	for _, task := range tasks {
		assert.Nil(t, s.LaunchAll(task))
	}

	strictSleep(max.Add(time.Second * 3))
	mlock.Lock()
	fmt.Println("total delay: ", last.Sub(max))
	assert.Equal(t, Retry*Total, counter, "tasks must be scheduled without missing!")
	mlock.Unlock()
}

func TestScheduleRecover(t *testing.T) {
	mlock.Lock()
	counter = 0
	mlock.Unlock()

	// Prepare task into data store

	start := time.Now().UTC()
	max := start
	for i := 0; i < Total; i++ {
		e := start.Add(time.Millisecond * time.Duration(rand.Intn(1000)))
		if e.After(max) {
			max = e
		}
		assert.Nil(t, store.Save(&ArbitraryTask{
			Info:      "hello world!",
			Retry:     false,
			id:        fmt.Sprintf("task%d", i),
			execution: e,
		}), "store must success!")
	}

	s := goscheduler.New()
	assert.Nil(t, s.RecoverAll(&ArbitraryTask{}), "recover must success!")

	strictSleep(max.Add(time.Second * 3))

	mlock.Lock()
	fmt.Println("total delay: ", last.Sub(max))
	assert.Equal(t, Retry*Total, counter, "tasks must be scheduled without missing!")
	mlock.Unlock()
}
