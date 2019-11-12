// Copyright 2018-2019 Changkun Ou. All rights reserved.
// Use of this source code is governed by a MIT
// license that can be found in the LICENSE file.

package sched

import (
	"encoding/json"
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

func TestSchedInitFail(t *testing.T) {
	_, err := Init("rdis://127.0.0.1:6323/123123")
	if err == nil {
		t.Fatal("Init with wrong format is sucess: ", err)
	}
}

func TestSchedMasiveSchedule(t *testing.T) {
	tests.O.Clear()
	Init("redis://127.0.0.1:6379/2")
	defer Stop()

	start := time.Now().UTC()
	expectedOrder := []string{}

	futures := make([]TaskFuture, 20)
	for i := 0; i < 20; i++ {
		key := fmt.Sprintf("task-%d", i)
		task := tests.NewTask(key, start.Add(time.Millisecond*10*time.Duration(i)))
		expectedOrder = append(expectedOrder, key)
		future, _ := Submit(task)
		futures[i] = future
	}
	for i := range futures {
		fmt.Println(futures[i].Get())
	}
	if !reflect.DeepEqual(expectedOrder, tests.O.Get()) {
		t.Errorf("execution order wrong, got: %v", tests.O.Get())
	}

}

func TestSchedRecover(t *testing.T) {
	tests.O.Clear()
	start := time.Now().UTC()
	Init("redis://127.0.0.1:6379/2")
	defer Stop()

	// save task into database
	want := []string{}
	for i := 0; i < 10; i++ {
		key := fmt.Sprintf("task-%d", i)
		task := tests.NewTask(key, start.Add(time.Millisecond*10*time.Duration(i)))
		want = append(want, key)
		if err := save(task); err != nil {
			t.Errorf("store with task-unique-id error: %v\n", err)
			return
		}
	}

	// recover back
	futures, err := sched0.recover(&tests.Task{})
	if err != nil {
		t.Errorf("recover task with task-unique-id error: %v\n", err)
		return
	}
	for i := range futures {
		fmt.Println(futures[i].Get())
	}
	if !reflect.DeepEqual(tests.O.Get(), want) {
		t.Errorf("recover execution order is not as expected, got: %v", tests.O.Get())
	}
}

func TestSchedSubmit(t *testing.T) {
	tests.O.Clear()
	start := time.Now().UTC()
	Init("redis://127.0.0.1:6379/2")
	defer Stop()

	// save task into database
	futures := make([]TaskFuture, 10)
	for i := 0; i < 10; i++ {
		key := fmt.Sprintf("task-%d", i)
		task := tests.NewRetryTask(key, start.Add(time.Millisecond*100*time.Duration(i)), 2)
		future, err := Submit(task)
		if err != nil {
			t.Fatalf("submit task failed: task %+v, err: %+v", task, err)
		}
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
	Init("redis://127.0.0.1:6379/2")
	defer Stop()

	task1 := tests.NewTask("task-1", start.Add(time.Millisecond*100))
	Submit(task1)
	task2 := tests.NewTask("task-2", start.Add(time.Millisecond*30))
	taskAreplica := tests.NewTask("task-1", start.Add(time.Millisecond*10))

	var wg sync.WaitGroup
	wg.Add(2)
	go func(t Task) {
		future, _ := Submit(t)
		future.Get()
		wg.Done()
	}(task2)
	go func(t Task) {
		future, _ := Trigger(t)
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
	Init("redis://127.0.0.1:6379/2")
	defer Stop()

	task1 := tests.NewTask("task-1", start.Add(time.Millisecond*100))
	Submit(task1)
	task2 := tests.NewTask("task-2", start.Add(time.Millisecond*30))

	var wg sync.WaitGroup
	wg.Add(2)
	go func(t Task) {
		future, _ := Submit(t)
		future.Get()
		wg.Done()
	}(task2)
	go func(t Task) {
		future, _ := Trigger(t)
		future.Get()
		wg.Done()
	}(task1)
	wg.Wait()

	want := []string{"task-1", "task-2"}
	if !reflect.DeepEqual(tests.O.Get(), want) {
		t.Errorf("launch task execution order is not as expected, got: %v", tests.O.Get())
	}
}

func set(key string, postpone time.Duration, t Task) {
	result, _ := sched0.cache.Get(prefixTask + key)
	r := &record{}
	json.Unmarshal([]byte(result), r)
	d, _ := json.Marshal(r.Data)
	temp := reflect.New(reflect.ValueOf(t).Elem().Type()).Interface().(Task)
	json.Unmarshal(d, &temp)
	temp.SetID(key)
	temp.SetExecution(r.Execution.Add(postpone))
	data, _ := json.Marshal(&record{
		ID:        temp.GetID(),
		Execution: temp.GetExecution(),
		Data:      temp,
	})
	sched0.cache.Set(prefixTask+temp.GetID(), string(data))
}

func TestSchedSchedule3(t *testing.T) {
	tests.O.Clear()
	start := time.Now().UTC()
	Init("redis://127.0.0.1:6379/2")
	defer Stop()

	// task1 with 1 sec later
	task1 := tests.NewTask("task-1", start.Add(time.Second))
	future, err := Submit(task1)
	if err != nil {
		t.FailNow()
	}

	// somehow override database time to 2 sec, the actual execution should be later
	set("task-1", time.Second*2, &tests.Task{})

	// sleep 1 sec
	strictSleep(start.Add(time.Second))

	// no execution at the moment
	want := []string{}
	if !reflect.DeepEqual(tests.O.Get(), want) {
		t.Errorf("submit task before execution is not as expected, got: %v", tests.O.Get())
	}

	fmt.Println(future.Get())

	// should have executed
	want = []string{"task-1"}
	if !reflect.DeepEqual(tests.O.Get(), want) {
		t.Errorf("submit task execution order is not as expected, got: %v", tests.O.Get())
	}
}

func TestSchedPause(t *testing.T) {
	tests.O.Clear()
	Init("redis://127.0.0.1:6379/2")
	defer Stop()

	start := time.Now().UTC()
	// task1 with 1 sec later
	task1 := tests.NewTask("task-1", start.Add(time.Second))
	future, _ := Submit(task1)

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
	Init("redis://127.0.0.1:6379/2")
	start := time.Now().UTC()
	// task1 with 1 sec later
	task1 := tests.NewTask("task-1", start.Add(time.Second))
	future, _ := Submit(task1)
	time.Sleep(time.Second + 500*time.Millisecond)
	Stop()
	fmt.Println(future.Get())
	want := []string{"task-1"}
	if !reflect.DeepEqual(tests.O.Get(), want) {
		t.Errorf("submit task execution order is not as expected, got: %v", tests.O.Get())
	}
}

func TestSchedPanic(t *testing.T) {
	tests.O.Clear()
	Init("redis://127.0.0.1:6379/2")
	defer Stop()

	start := time.Now().UTC()
	// task1 with 1 sec later
	task1 := tests.NewPanicTask("task-1", start.Add(time.Second))
	future, _ := Submit(task1)
	time.Sleep(time.Second + 100*time.Millisecond)
	sched0.cache.Del(prefixTask + "task-1")
	Stop()
	fmt.Println(future.Get())
	want := []string{"task-1"}
	if !reflect.DeepEqual(tests.O.Get(), want) {
		t.Errorf("submit task execution order is not as expected, got: %v", tests.O.Get())
	}
}

func TestSchedRecoverFail(t *testing.T) {
	tests.O.Clear()
	start := time.Now().UTC()
	url := "redis://127.0.0.1:6379/2"
	Init(url)
	task := tests.NewNonExportTask("task-0", start.Add(time.Millisecond*10))
	if err := save(task); err != nil {
		t.Errorf("store with task-unique-id error: %v\n", err)
		return
	}
	sched0.cache.Close()

	futures, err := Init(url, &tests.Task{})
	if err != nil {
		t.Errorf("recover task with task-unique-id error: %v\n", err)
		return
	}
	defer Stop()

	for i := range futures {
		futures[i].Get()
	}
	if !reflect.DeepEqual(tests.O.Get(), []string{}) {
		t.Errorf("recover execution order is not as expected, got: %v", tests.O.Get())
	}

	sched0.cache.Del(prefixTask + "task-0")
}

func TestSchedError(t *testing.T) {
	Init("redis://127.0.0.1:6379/2")
	sched0.cache.Close()
	if _, err := sched0.recover(&tests.Task{}); err == nil {
		t.Fatalf("recover without conn must error, got nil")
	}
	if _, err := sched0.submit(&tests.Task{}); err == nil {
		t.Fatalf("submit without conn must error, got nil")
	}
	if _, err := sched0.trigger(&tests.Task{}); err == nil {
		t.Fatalf("trigger without conn must error, got nil")
	}
	if _, err := sched0.verify(&tests.Task{}); err == nil {
		t.Fatalf("verify without conn must error, got nil")
	}
	r := record{
		ID:        "id",
		Execution: time.Now(),
		Data:      func() {},
	}
	if err := r.save(); err == nil {
		t.Fatalf("save func marshal must error, got nil")
	}
	sched0 = &sched{
		timer: unsafe.Pointer(time.NewTimer(0)),
		tasks: newTaskQueue(),
		cache: sched0.cache,
	}
	sched0.worker()
	sched0.arrival(newTaskItem(&tests.Task{}))
	sched0.execute(newTaskItem(&tests.Task{}))
	Pause()
	sched0.worker()
	Resume()

	Init("redis://127.0.0.1:6379/2")
	r = record{
		ID:        "error",
		Execution: time.Now(),
		Data:      &tests.RetryTask{RetryCount: 1},
	}
	if err := r.save(); err != nil {
		t.Fatalf("save object must not nil, got %v", err)
	}
	sched0.cache.Close()
	sched0.load("error", &tests.Task{})
	Init("redis://127.0.0.1:6379/2")
	sched0.cache.Del(prefixTask + "error")
	sched0.cache.Close()
}

func TestSchedStop2(t *testing.T) {
	sched0 = &sched{
		timer: unsafe.Pointer(time.NewTimer(0)),
		tasks: newTaskQueue(),
	}
	Init("redis://127.0.0.1:6379/2")
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
