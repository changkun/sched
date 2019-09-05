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
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		t1 := tests.NewTask(fmt.Sprintf("task-%d", i), time.Now().Add(time.Millisecond))
		t2 := tests.NewTask(fmt.Sprintf("task-%d", i+1), time.Now().Add(time.Millisecond))
		t3 := tests.NewTask(fmt.Sprintf("task-%d", i+2), time.Now().Add(time.Second))
		t4 := tests.NewTask(fmt.Sprintf("task-%d", i+3), time.Now().Add(time.Second))
		b.StartTimer()

		_ = Submit(t1)
		_ = Submit(t2)
		_ = Submit(t3)
		_ = Submit(t4)
		_ = Trigger(t3)
		_ = Trigger(t4)
		Wait()
	}
}
