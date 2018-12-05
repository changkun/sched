// Copyright 2018 Changkun Ou. All rights reserved.
// Use of this source code is governed by a MIT
// license that can be found in the LICENSE file.

package sched

import (
	"encoding/json"
	"fmt"
	"reflect"
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

// isTaskScheduled checks if all tasks are scheduled
func isTaskScheduled() error {
	keys, err := getRecords()
	if err != nil {
		return err
	}
	if len(keys) != 0 {
		return fmt.Errorf("There are tasks unschedued: %v", keys)
	}
	return nil
}

func TestSchedMasiveSchedule(t *testing.T) {
	tests.O.Clear()
	Init("redis://127.0.0.1:6379/2")
	defer Stop()

	start := time.Now().UTC()
	expectedOrder := []string{}
	for i := 0; i < 20; i++ {
		key := fmt.Sprintf("task-%d", i)
		task := tests.NewTask(key, start.Add(time.Millisecond*10*time.Duration(i)))
		expectedOrder = append(expectedOrder, key)
		Submit(task)
	}
	strictSleep(start.Add(time.Millisecond * 10 * 25))

	if err := isTaskScheduled(); err != nil {
		t.Error("There are tasks unscheduled")
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
	if err := sched0.recover(&tests.Task{}); err != nil {
		t.Errorf("recover task with task-unique-id error: %v\n", err)
		return
	}
	strictSleep(start.Add(time.Second))
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
	for i := 0; i < 10; i++ {
		key := fmt.Sprintf("task-%d", i)
		task := tests.NewRetryTask(key, start.Add(time.Millisecond*10*time.Duration(i)), 2)
		if err := Submit(task); err != nil {
			t.Fatal("submit task fail:", task)
		}
	}
	want := []string{
		"task-0", "task-1", "task-2", "task-3", "task-4",
		"task-0", "task-5", "task-1", "task-6", "task-2",
		"task-7", "task-3", "task-8", "task-4", "task-0",
		"task-9", "task-5", "task-1", "task-6", "task-2",
		"task-7", "task-3", "task-8", "task-4", "task-9",
		"task-5", "task-6", "task-7", "task-8", "task-9",
	}
	strictSleep(start.Add(time.Second))
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
	go Submit(task2)
	go Trigger(taskAreplica)

	strictSleep(start.Add(time.Second))
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
	go Submit(task2)
	go Trigger(task1)

	strictSleep(start.Add(time.Second))
	want := []string{"task-1", "task-2"}
	if !reflect.DeepEqual(tests.O.Get(), want) {
		t.Errorf("launch task execution order is not as expected, got: %v", tests.O.Get())
	}
}

func set(key string, postpone time.Duration, t Task) {
	result, _ := cache0.GET(prefixTask + key)
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
	cache0.SET(prefixTask+temp.GetID(), string(data))
}

func TestSchedSchedule3(t *testing.T) {
	tests.O.Clear()
	start := time.Now().UTC()
	Init("redis://127.0.0.1:6379/2")
	defer Stop()

	// task1 with 1 sec later
	task1 := tests.NewTask("task-1", start.Add(time.Second))
	Submit(task1)

	// somehow override database time to 2 sec, the actual execution should be later
	set("task-1", time.Second*2, &tests.Task{})

	// sleep 1 sec
	strictSleep(start.Add(time.Second))

	// no execution at the moment
	want := []string{}
	if !reflect.DeepEqual(tests.O.Get(), want) {
		t.Errorf("submit task before execution is not as expected, got: %v", tests.O.Get())
	}

	// sleep 2 sec
	strictSleep(start.Add(time.Second * 3))

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
	Submit(task1)

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
	// sleep until start+3sec
	strictSleep(start.Add(time.Second * 3))
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
	Submit(task1)
	time.Sleep(time.Second + 500*time.Millisecond)
	Stop()
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
	Submit(task1)
	time.Sleep(time.Second + 100*time.Millisecond)
	cache0.DEL(prefixTask + "task-1")
	Stop()
	want := []string{"task-1"}
	if !reflect.DeepEqual(tests.O.Get(), want) {
		t.Errorf("submit task execution order is not as expected, got: %v", tests.O.Get())
	}
}

func TestSchedRecoverFail(t *testing.T) {
	tests.O.Clear()
	start := time.Now().UTC()
	url := "redis://127.0.0.1:6379/2"
	connectCache(url)
	task := tests.NewNonExportTask("task-0", start.Add(time.Millisecond*10))
	if err := save(task); err != nil {
		t.Errorf("store with task-unique-id error: %v\n", err)
		return
	}
	cache0.Close()

	if err := Init(url, &tests.Task{}); err != nil {
		t.Errorf("recover task with task-unique-id error: %v\n", err)
		return
	}
	defer Stop()
	strictSleep(start.Add(time.Second))
	if !reflect.DeepEqual(tests.O.Get(), []string{}) {
		t.Errorf("recover execution order is not as expected, got: %v", tests.O.Get())
	}

	cache0.DEL(prefixTask + "task-0")
}

func TestSchedError(t *testing.T) {
	connectCache("redis://127.0.0.1:6379/2")
	cache0.Close()
	if err := sched0.recover(&tests.Task{}); err == nil {
		t.Fatalf("recover without conn must error, got nil")
	}
	if err := sched0.submit(&tests.Task{}); err == nil {
		t.Fatalf("submit without conn must error, got nil")
	}
	if err := sched0.trigger(&tests.Task{}); err == nil {
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
	}
	sched0.worker()
	sched0.arrival(&tests.Task{})
	sched0.execute(&tests.Task{})
	sched0.retry(&tests.Task{})
	Pause()
	sched0.worker()
	Resume()

	connectCache("redis://127.0.0.1:6379/2")
	r = record{
		ID:        "error",
		Execution: time.Now(),
		Data:      &tests.RetryTask{RetryCount: 1},
	}
	if err := r.save(); err != nil {
		t.Fatalf("save object must not nil, got %v", err)
	}
	cache0.Close()
	sched0.load("error", &tests.Task{})
	connectCache("redis://127.0.0.1:6379/2")
	cache0.DEL(prefixTask + "error")
	cache0.Close()
}

func TestSchedStop2(t *testing.T) {
	sched0 = &sched{
		timer: unsafe.Pointer(time.NewTimer(0)),
		tasks: newTaskQueue(),
	}
	connectCache("redis://127.0.0.1:6379/2")
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
