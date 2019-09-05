package simsched

import (
	"fmt"
	"reflect"
	"sync"
	"sync/atomic"
	"testing"
	"time"
	"unsafe"

	"github.com/changkun/sched/tests"
)

// sleep to wait execution, a strict wait tolerance: 100 milliseconds
func strictSleep(latest time.Time) {
	time.Sleep(latest.Sub(time.Now().UTC()) + time.Millisecond*100)
}

func TestSchedMasiveSchedule(t *testing.T) {
	tests.O.Clear()
	defer Stop()

	start := time.Now().UTC()
	expectedOrder := []string{}

	futures := make([]TaskFuture, 20)
	for i := 0; i < 20; i++ {
		key := fmt.Sprintf("task-%d", i)
		task := tests.NewTask(key, start.Add(time.Millisecond*10*time.Duration(i)))
		expectedOrder = append(expectedOrder, key)
		future := Submit(task)
		futures[i] = future
	}
	for i := range futures {
		fmt.Println(futures[i].Get())
	}
	if !reflect.DeepEqual(expectedOrder, tests.O.Get()) {
		t.Errorf("execution order wrong, got: %v", tests.O.Get())
	}

}

func TestSchedSubmit(t *testing.T) {
	tests.O.Clear()
	start := time.Now().UTC()
	defer Stop()

	// save task into database
	futures := make([]TaskFuture, 10)
	for i := 0; i < 10; i++ {
		key := fmt.Sprintf("task-%d", i)
		task := tests.NewRetryTask(key, start.Add(time.Millisecond*100*time.Duration(i)), 2)
		future := Submit(task)
		futures[i] = future
	}
	want := []string{
		"task-0", "task-1", "task-2", "task-3", "task-4",
		"task-0", "task-5", "task-1", "task-6", "task-2",
		"task-7", "task-3", "task-8", "task-4", "task-0",
		"task-9", "task-5", "task-1", "task-6", "task-2",
		"task-7", "task-3", "task-8", "task-4", "task-9",
		"task-5", "task-6", "task-7", "task-8", "task-9",
	}
	for i := range futures {
		fmt.Printf("%v: %v\n", i, futures[i].Get())
	}
	if !reflect.DeepEqual(len(tests.O.Get()), len(want)) {
		t.Errorf("submit retry task execution order is not as expected, want %d, got: %d", len(want), len(tests.O.Get()))
	}
}

func TestSchedSchedule1(t *testing.T) {
	tests.O.Clear()
	start := time.Now().UTC()
	defer Stop()

	task1 := tests.NewTask("task-1", start.Add(time.Millisecond*100))
	Submit(task1)
	task2 := tests.NewTask("task-2", start.Add(time.Millisecond*30))
	taskAreplica := tests.NewTask("task-1", start.Add(time.Millisecond*10))

	var wg sync.WaitGroup
	wg.Add(2)
	go func(t Task) {
		future := Submit(t)
		future.Get()
		wg.Done()
	}(task2)
	go func(t Task) {
		future := Trigger(t)
		future.Get()
		wg.Done()
	}(taskAreplica)
	wg.Wait()
	want := []string{"task-1", "task-2"}
	if !reflect.DeepEqual(tests.O.Get(), want) {
		t.Errorf("launch task execution order is not as expected, got: %v", tests.O.Get())
	}
}

func TestSchedSchedule2(t *testing.T) {
	tests.O.Clear()
	start := time.Now().UTC()
	defer Stop()

	task1 := tests.NewTask("task-1", start.Add(time.Millisecond*100))
	Submit(task1)
	task2 := tests.NewTask("task-2", start.Add(time.Millisecond*30))

	var wg sync.WaitGroup
	wg.Add(2)
	go func(t Task) {
		future := Submit(t)
		future.Get()
		wg.Done()
	}(task2)
	go func(t Task) {
		future := Trigger(t)
		future.Get()
		wg.Done()
	}(task1)
	wg.Wait()

	want := []string{"task-1", "task-2"}
	if !reflect.DeepEqual(tests.O.Get(), want) {
		t.Errorf("launch task execution order is not as expected, got: %v", tests.O.Get())
	}
}

func TestSchedPause(t *testing.T) {
	tests.O.Clear()
	defer Stop()

	start := time.Now().UTC()
	// task1 with 1 sec later
	task1 := tests.NewTask("task-1", start.Add(time.Second))
	future := Submit(task1)

	// pause sched and sleep 1 sec, task1 should not be executed
	Pause()
	time.Sleep(time.Second * 2)
	want := []string{}
	if !reflect.DeepEqual(tests.O.Get(), want) {
		t.Errorf("submit task execution order is not as expected, got: %v", tests.O.Get())
	}

	// at this moment, task-1 should be executed asap
	// should have executed
	Resume()

	fmt.Println(future.Get())
	want = []string{"task-1"}
	if !reflect.DeepEqual(tests.O.Get(), want) {
		t.Errorf("submit task execution order is not as expected, got: %v", tests.O.Get())
	}
}

func TestSchedStop(t *testing.T) {
	tests.O.Clear()
	start := time.Now().UTC()
	// task1 with 1 sec later
	task1 := tests.NewTask("task-1", start.Add(time.Second))
	future := Submit(task1)
	time.Sleep(time.Second + 500*time.Millisecond)
	Stop()
	fmt.Println(future.Get())
	want := []string{"task-1"}
	if !reflect.DeepEqual(tests.O.Get(), want) {
		t.Errorf("submit task execution order is not as expected, got: %v", tests.O.Get())
	}
}

func TestSchedError(t *testing.T) {
	sched0 = &sched{
		timer: unsafe.Pointer(time.NewTimer(0)),
		tasks: newTaskQueue(),
	}
	sched0.worker()
	sched0.arrival(newTaskItem(&tests.Task{}))
	sched0.execute(newTaskItem(&tests.Task{}))
	Pause()
	sched0.worker()
	Resume()
}

func TestSchedStop2(t *testing.T) {
	sched0 = &sched{
		timer: unsafe.Pointer(time.NewTimer(0)),
		tasks: newTaskQueue(),
	}
	atomic.AddUint64(&sched0.running, 2)
	sign1 := make(chan int, 1)
	sign2 := make(chan int, 1)
	go func() {
		Stop()
		sign1 <- 1
		close(sign1)
	}()
	go func() {
		time.Sleep(time.Millisecond * 100)
		atomic.AddUint64(&sched0.running, ^uint64(0))
		time.Sleep(time.Millisecond * 100)
		atomic.AddUint64(&sched0.running, ^uint64(0))
		sign2 <- 2
	}()

	order := []int{}
	select {
	case n := <-sign1:
		order = append(order, n)
	case n := <-sign2:
		order = append(order, n)
	}
	select {
	case n := <-sign1:
		order = append(order, n)
	case n := <-sign2:
		order = append(order, n)
	}
	want := []int{2, 1}

	if !reflect.DeepEqual(order, want) {
		t.Fatalf("unexpected order of stop")
	}

}

func BenchmarkSubmit(b *testing.B) {
	// go test -bench=BenchmarkSubmit -run=^$ -cpuprofile cpu.prof -memprofile mem.prof -trace trace.out
	// go tool pprof -http=:8080 cpu.prof
	// go tool pprof -http=:8080 mem.prof
	// go tool trace trace.out
	for size := 1; size < 10; size += 100 {
		println("size: ", size)
		b.Run(fmt.Sprintf("#tasks-%d", size), func(b *testing.B) {
			ts := newTasks(size)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				// prepare problem size
				b.StopTimer()
				for j := 0; j < size-1; j++ {
					Submit(ts[j])
				}
				b.StartTimer()

				// enqueue under the problem size
				Submit(ts[size-1])
				Wait()
			}
		})
	}
}

func newTasks(size int) []Task {
	ts := make([]Task, size)
	for i := 0; i < size; i++ {
		ts[i] = newZeroTask(fmt.Sprintf("task-%d", i), time.Now().Add(time.Millisecond))
	}
	return ts
}

// tt implements tt.Interface
type tt struct {
	Public    string
	id        string
	execution time.Time
}

// newZeroTask creates a task
func newZeroTask(id string, e time.Time) *tt {
	return &tt{
		Public:    "not nil",
		id:        id,
		execution: e,
	}
}

// GetID get task id
func (t *tt) GetID() (id string) {
	id = t.id
	return
}

// GetExecution get execution time
func (t *tt) GetExecution() (execute time.Time) {
	execute = t.execution
	return
}

// GetTimeout get timeout of execution
func (t *task) GetTimeout() (executeTimeout time.Duration) {
	return time.Second
}

// GetRetryTime get retry execution duration
func (t *tt) GetRetryTime() time.Time {
	return time.Now().UTC().Add(time.Second)
}

// SetID sets the id of a task
func (t *tt) SetID(id string) {
	t.id = id
}

// IsValidID check id is valid
func (t *tt) IsValidID() bool {
	return true
}

// SetExecution sets the execution time of a task
func (t *tt) SetExecution(current time.Time) (old time.Time) {
	old = t.execution
	t.execution = current
	return
}

// Execute is the actual execution block
func (t *tt) Execute() (result interface{}, retry bool, fail error) {
	result = 1 // avoid allocation
	return
}
