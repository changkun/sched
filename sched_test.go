package sched_test

import (
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/changkun/sched"
	"github.com/changkun/sched/internal/pool"
	"github.com/changkun/sched/internal/store"
	"github.com/changkun/sched/tests"
)

// // ArbitraryTask implements task.Interface that can be scheduled by schedule
// type ArbitraryTask struct {
// 	Info  string
// 	Retry bool
// 	Count int

// 	id        string
// 	execution time.Time
// }

// // ID implements a task can be scheduled by scheduler
// func (c *ArbitraryTask) GetID() string {
// 	return c.id
// }
// func (c *ArbitraryTask) SetID(id string) {
// 	c.id = id
// }

// func (c *ArbitraryTask) GetExecution() time.Time {
// 	return c.execution
// }

// func (c *ArbitraryTask) GetTimeout() time.Duration {
// 	return time.Second
// }

// func (c *ArbitraryTask) GetRetryDuration() time.Duration {
// 	return time.Second
// }

// func (c *ArbitraryTask) SetExecution(t time.Time) (old time.Time) {
// 	old = c.execution
// 	c.execution = t
// 	return
// }

// const (
// 	Total = 100
// 	Retry = 3
// )

// type Counter struct {
// 	first   time.Time
// 	last    time.Time
// 	mark    int
// 	counter int
// 	mu      sync.Mutex
// }

// func (m *Counter) SetFirst(f time.Time) {
// 	m.mu.Lock()
// 	defer m.mu.Unlock()

// 	m.first = f
// }

// func (m *Counter) GetFirst() time.Time {
// 	m.mu.Lock()
// 	defer m.mu.Unlock()

// 	return m.first
// }

// func (m *Counter) SetLast(l time.Time) {
// 	m.mu.Lock()
// 	defer m.mu.Unlock()

// 	m.last = l
// }
// func (m *Counter) GetLast() time.Time {
// 	m.mu.Lock()
// 	defer m.mu.Unlock()

// 	return m.last
// }

// func (m *Counter) GetMark() int {
// 	m.mu.Lock()
// 	defer m.mu.Unlock()

// 	return m.mark
// }
// func (m *Counter) SetMark(num int) {
// 	m.mu.Lock()
// 	defer m.mu.Unlock()

// 	m.mark = num
// }
// func (m *Counter) AddCount() {
// 	m.mu.Lock()
// 	defer m.mu.Unlock()

// 	m.counter++
// }
// func (m *Counter) GetCount() int {
// 	m.mu.Lock()
// 	defer m.mu.Unlock()

// 	return m.counter
// }

// var counter Counter

// func (c *ArbitraryTask) Execute() (bool, error) {
// 	if c.Count >= Retry {
// 		return false, nil
// 	}

// 	counter.SetLast(time.Now().UTC())
// 	counter.AddCount()

// 	// fmt.Printf("ArbitraryTask %s is scheduled: %s, count: %d, execution tolerance: %s\n", c.id, c.Info, c.Count, time.Now().UTC().Sub(c.GetExecution()))
// 	c.Count++
// 	return true, nil
// }

// sleep to wait execution, a strict wait tolerance: 100 milliseconds
func strictSleep(latest time.Time) {
	time.Sleep(latest.Sub(time.Now().UTC()) + time.Millisecond*100)
}

func init() {
	sched.Init("redis://127.0.0.1:6379/1")
}

// func TestScheduleSetup(t *testing.T) {
// 	counter = Counter{}

// 	start := time.Now().UTC()
// 	max := start
// 	tasks := []*ArbitraryTask{}
// 	for i := 0; i < Total; i++ {
// 		e := start.Add(time.Millisecond * time.Duration(rand.Intn(1000)))
// 		if e.After(max) {
// 			max = e
// 		}
// 		tasks = append(tasks, &ArbitraryTask{
// 			Info:      "hello world!",
// 			Retry:     false,
// 			id:        fmt.Sprintf("task%d", i),
// 			execution: e, // add no more than 10s
// 		})

// 	}
// 	s := sched.New()
// 	for _, task := range tasks {
// 		assert.Nil(t, s.Setup(task))
// 	}

// 	strictSleep(max.Add(time.Second * 3))
// 	fmt.Println("total delay: ", counter.GetLast().Sub(max))
// 	fmt.Println("last required schedule: ", max)
// 	fmt.Println("actual schedule:        ", counter.GetLast())
// 	assert.Equal(t, Retry*Total, counter.GetCount(), "tasks must be scheduled without missing!")
// }

// func TestScheduleLaunch(t *testing.T) {
// 	counter = Counter{}

// 	start := time.Now().UTC()
// 	max := start
// 	tasks := []*ArbitraryTask{}
// 	for i := 0; i < Total; i++ {
// 		e := start.Add(time.Millisecond * time.Duration(rand.Intn(1000)))
// 		if e.After(max) {
// 			max = e
// 		}
// 		tasks = append(tasks, &ArbitraryTask{
// 			Info:      "hello world!",
// 			Retry:     false,
// 			id:        fmt.Sprintf("task%d", i),
// 			execution: e,
// 		})
// 	}

// 	s := sched.New()

// 	// Schedule all first
// 	for _, task := range tasks {
// 		assert.Nil(t, s.Setup(task))
// 	}

// 	// Launch all by then
// 	for _, task := range tasks {
// 		assert.Nil(t, s.Launch(task))
// 	}

// 	strictSleep(max.Add(time.Second * 3))
// 	fmt.Println("total delay: ", counter.GetLast().Sub(max))
// 	assert.Equal(t, Retry*Total, counter.GetCount(), "tasks must be scheduled without missing!")
// }

// func TestScheduleRecover(t *testing.T) {
// 	counter = Counter{}

// 	// Prepare task into data store

// 	start := time.Now().UTC()
// 	max := start
// 	for i := 0; i < Total; i++ {
// 		e := start.Add(time.Millisecond * time.Duration(rand.Intn(1000)))
// 		if e.After(max) {
// 			max = e
// 		}
// 		assert.Nil(t, store.Save(&ArbitraryTask{
// 			Info:      "hello world!",
// 			Retry:     false,
// 			id:        fmt.Sprintf("task%d", i),
// 			execution: e,
// 		}), "store must success!")
// 	}

// 	s := sched.New()
// 	assert.Nil(t, s.Recover(&ArbitraryTask{}), "recover must success!")

// 	strictSleep(max.Add(time.Second * 2))

// 	fmt.Println("total delay: ", counter.GetLast().Sub(max))
// 	assert.Equal(t, Retry*Total, counter.GetCount(), "tasks must be scheduled without missing!")
// }

// type TaskA struct {
// 	Info string

// 	id  string
// 	end time.Time
// }

// // ID implements a task can be scheduled by scheduler
// func (c *TaskA) GetID() string {
// 	return c.id
// }
// func (c *TaskA) SetID(id string) {
// 	c.id = id
// }

// func (c *TaskA) GetExecution() time.Time {
// 	return c.end
// }

// func (c *TaskA) GetTimeout() time.Duration {
// 	return time.Second * 10
// }

// func (c *TaskA) GetRetryDuration() time.Duration {
// 	return time.Second * 1
// }

// func (c *TaskA) SetExecution(t time.Time) time.Time {
// 	old := c.end
// 	c.end = t
// 	return old
// }

// func (c *TaskA) Execute() (bool, error) {
// 	counter.SetMark(1)
// 	counter.AddCount()
// 	fmt.Printf("ArbitraryTask %s is scheduled: %s\n", c.id, c.Info)
// 	return false, nil
// }

// type TaskB struct {
// 	Info string

// 	id  string
// 	end time.Time
// }

// // ID implements a task can be scheduled by scheduler
// func (c *TaskB) GetID() string {
// 	return c.id
// }
// func (c *TaskB) SetID(id string) {
// 	c.id = id
// }

// func (c *TaskB) GetExecution() time.Time {
// 	return c.end
// }

// func (c *TaskB) GetTimeout() time.Duration {
// 	return time.Second * 10
// }

// func (c *TaskB) GetRetryDuration() time.Duration {
// 	return time.Second * 1
// }

// func (c *TaskB) SetExecution(t time.Time) time.Time {
// 	old := c.end
// 	c.end = t
// 	return old
// }

// func (c *TaskB) Execute() (bool, error) {
// 	counter.SetMark(2)
// 	counter.AddCount()
// 	fmt.Printf("TaskB %s is scheduled: %s\n", c.id, c.Info)
// 	return false, nil
// }

// func TestComplexSchedule1(t *testing.T) {
// 	counter = Counter{}
// 	start := time.Now().UTC()
// 	s := sched.New()

// 	taskA := &TaskA{
// 		Info: "hello world!",
// 		id:   "taskA",
// 		end:  start.Add(time.Second * 10),
// 	}
// 	assert.Nil(t, s.Setup(taskA), "Setup task A must be success!")
// 	time.Sleep(time.Second * 2)
// 	taskB := &TaskB{
// 		Info: "hello world!",
// 		id:   "taskB",
// 		end:  start.Add(time.Second * 2),
// 	}
// 	go s.Setup(taskB)  //assert.Nil(t, , "Launch task B immediately must be success!")
// 	go s.Launch(taskA) //assert.Nil(t, , "Launch task A immediately must be success!")

// 	strictSleep(start.Add(time.Second * 3))

// 	assert.True(t, 1 == counter.GetMark() || 2 == counter.GetMark())
// 	assert.Equal(t, 2, counter.GetCount())
// }

// func TestComplexSchedule2(t *testing.T) {
// 	counter = Counter{}
// 	start := time.Now().UTC()
// 	s := sched.New()

// 	taskA := &TaskA{
// 		Info: "hello world!",
// 		id:   "taskA",
// 		end:  start.Add(time.Second * 10),
// 	}

// 	assert.Nil(t, s.Setup(taskA), "Setup task A must be success!")

// 	time.Sleep(time.Second * 2)

// 	taskB := &TaskB{
// 		Info: "hello world!",
// 		id:   "taskB",
// 		end:  start.Add(time.Second * 2),
// 	}
// 	go s.Setup(taskB)  //assert.Nil(t, , "Launch task B immediately must be success!")
// 	go s.Launch(taskA) //assert.Nil(t, , "Launch task A immediately must be success!")

// 	strictSleep(start.Add(time.Second * 3))

// 	assert.True(t, 1 == counter.GetMark() || 2 == counter.GetMark())
// 	assert.Equal(t, 2, counter.GetCount())
// }

// func set(key string, postpone time.Duration, t task.Interface) {
// 	conn := pool.Get()
// 	defer conn.Close()

// 	result, _ := redis.String(conn.Do("GET", "sched:task:"+key))
// 	r := &store.Record{}
// 	json.Unmarshal([]byte(result), r)
// 	d, _ := json.Marshal(r.Data)
// 	temp := reflect.New(reflect.ValueOf(t).Elem().Type()).Interface().(task.Interface)
// 	json.Unmarshal(d, &temp)
// 	temp.SetID(key)
// 	temp.SetExecution(r.Execution.Add(postpone))
// 	data, _ := json.Marshal(&store.Record{
// 		ID:        temp.GetID(),
// 		Execution: temp.GetExecution(),
// 		Data:      temp,
// 	})
// 	conn.Do("SET", "sched:task:"+temp.GetID(), string(data))
// }
// func TestComplexSchedule3(t *testing.T) {
// 	counter = Counter{}
// 	start := time.Now().UTC()
// 	s := sched.New()

// 	taskA := &TaskA{
// 		Info: "hello world!",
// 		id:   "taskA",
// 		end:  start.Add(time.Second * 3),
// 	}
// 	assert.Nil(t, s.Setup(taskA), "Setup task A must be success!")
// 	var temp TaskA
// 	set("taskA", time.Second*5, &temp)
// 	strictSleep(start.Add(time.Second * 8))
// 	assert.Equal(t, 1, counter.GetCount())
// }

// isTaskScheduled checks if all tasks are scheduled
func isTaskScheduled() error {
	conn := pool.Get()
	defer conn.Close()

	keys, err := store.GetRecords()
	if err != nil {
		return err
	}
	if len(keys) != 0 {
		return fmt.Errorf("There are tasks unschedued: %v", keys)
	}
	return nil
}

func TestMasiveSchedule(t *testing.T) {
	tests.O.Clear()

	start := time.Now().UTC()
	s := sched.New()

	expectedOrder := []string{}
	for i := 0; i < 20; i++ {
		key := fmt.Sprintf("task-%d", i)
		task := tests.NewTask(key, start.Add(time.Millisecond*10*time.Duration(i)))
		expectedOrder = append(expectedOrder, key)
		s.Setup(task)
	}
	strictSleep(start.Add(time.Millisecond * 10 * 25))

	if err := isTaskScheduled(); err != nil {
		t.Error("There are tasks unscheduled")
	}

	if !reflect.DeepEqual(expectedOrder, tests.O.Get()) {
		t.Errorf("execution order wrong, got: %v", tests.O.Get())
	}
}
