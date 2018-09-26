// Copyright 2018 Changkun Ou. All rights reserved.
// Use of this source code is governed by a MIT
// license that can be found in the LICENSE file.

package sched_test

import (
	"encoding/json"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/changkun/sched"
	"github.com/changkun/sched/internal/pool"
	"github.com/changkun/sched/internal/store"
	"github.com/changkun/sched/task"
	"github.com/changkun/sched/tests"
	"github.com/gomodule/redigo/redis"
)

func init() {
	sched.Init("redis://127.0.0.1:6379/1")
}

// sleep to wait execution, a strict wait tolerance: 100 milliseconds
func strictSleep(latest time.Time) {
	time.Sleep(latest.Sub(time.Now().UTC()) + time.Millisecond*100)
}

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

func TestNew(t *testing.T) {
	if s := sched.New(); s == nil {
		t.Error("new scheduler fail!")
	}
}

func TestRecover(t *testing.T) {
	tests.O.Clear()
	start := time.Now().UTC()

	// save task into database
	want := []string{}
	for i := 0; i < 10; i++ {
		key := fmt.Sprintf("task-%d", i)
		task := tests.NewTask(key, start.Add(time.Millisecond*10*time.Duration(i)))
		want = append(want, key)
		if err := store.Save(task); err != nil {
			t.Errorf("store with task-unique-id error: %v\n", err)
			return
		}
	}

	// recover back
	if err := sched.Recover(&tests.Task{}); err != nil {
		t.Errorf("recover task with task-unique-id error: %v\n", err)
		return
	}
	strictSleep(start.Add(time.Second))
	if !reflect.DeepEqual(tests.O.Get(), want) {
		t.Errorf("recover execution order is not as expected, got: %v", tests.O.Get())
	}
}

func TestSetup(t *testing.T) {
	tests.O.Clear()
	start := time.Now().UTC()

	// save task into database
	for i := 0; i < 10; i++ {
		key := fmt.Sprintf("task-%d", i)
		task := tests.NewRetryTask(key, start.Add(time.Millisecond*10*time.Duration(i)), 2)
		if err := sched.Setup(task); err != nil {
			t.Fatal("setup task fail:", task)
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
	if !reflect.DeepEqual(tests.O.Get(), want) {
		t.Errorf("setup retry task execution order is not as expected, got: %v", tests.O.Get())
	}
}

func TestRecoverFail(t *testing.T) {
	tests.O.Clear()
	start := time.Now().UTC()
	task := tests.NewNonExportTask("task-0", start.Add(time.Millisecond*10))
	if err := store.Save(task); err != nil {
		t.Errorf("store with task-unique-id error: %v\n", err)
		return
	}
	if err := sched.Recover(&tests.Task{}); err != nil {
		t.Errorf("recover task with task-unique-id error: %v\n", err)
		return
	}
	strictSleep(start.Add(time.Millisecond * 100))
	if !reflect.DeepEqual(tests.O.Get(), []string{}) {
		t.Errorf("recover execution order is not as expected, got: %v", tests.O.Get())
	}
}

func TestSchedule1(t *testing.T) {
	tests.O.Clear()
	start := time.Now().UTC()

	task1 := tests.NewTask("task-1", start.Add(time.Millisecond*100))
	sched.Setup(task1)
	task2 := tests.NewTask("task-2", start.Add(time.Millisecond*30))
	taskAreplica := tests.NewTask("task-1", start.Add(time.Millisecond*10))
	go sched.Setup(task2)
	go sched.Launch(taskAreplica)

	strictSleep(start.Add(time.Second))
	want := []string{"task-1", "task-2"}
	if !reflect.DeepEqual(tests.O.Get(), want) {
		t.Errorf("launch task execution order is not as expected, got: %v", tests.O.Get())
	}
}
func TestSchedule2(t *testing.T) {
	tests.O.Clear()
	start := time.Now().UTC()

	task1 := tests.NewTask("task-1", start.Add(time.Millisecond*100))
	sched.Setup(task1)
	task2 := tests.NewTask("task-2", start.Add(time.Millisecond*30))
	go sched.Setup(task2)
	go sched.Launch(task1)

	strictSleep(start.Add(time.Second))
	want := []string{"task-1", "task-2"}
	if !reflect.DeepEqual(tests.O.Get(), want) {
		t.Errorf("launch task execution order is not as expected, got: %v", tests.O.Get())
	}
}

func set(key string, postpone time.Duration, t task.Interface) {
	conn := pool.Get()
	defer conn.Close()

	result, _ := redis.String(conn.Do("GET", "sched:task:"+key))
	r := &store.Record{}
	json.Unmarshal([]byte(result), r)
	d, _ := json.Marshal(r.Data)
	temp := reflect.New(reflect.ValueOf(t).Elem().Type()).Interface().(task.Interface)
	json.Unmarshal(d, &temp)
	temp.SetID(key)
	temp.SetExecution(r.Execution.Add(postpone))
	data, _ := json.Marshal(&store.Record{
		ID:        temp.GetID(),
		Execution: temp.GetExecution(),
		Data:      temp,
	})
	conn.Do("SET", "sched:task:"+temp.GetID(), string(data))
}

func TestSchedule3(t *testing.T) {
	tests.O.Clear()
	start := time.Now().UTC()

	// task1 with 1 sec later
	task1 := tests.NewTask("task-1", start.Add(time.Second))
	sched.Setup(task1)

	// somehow override database time to 2 sec, the actual execution should be later
	set("task-1", time.Second*2, &tests.Task{})

	// sleep 1 sec
	strictSleep(start.Add(time.Second))

	// no execution at the moment
	want := []string{}
	if !reflect.DeepEqual(tests.O.Get(), want) {
		t.Errorf("setup task before execution is not as expected, got: %v", tests.O.Get())
	}

	// sleep 2 sec
	strictSleep(start.Add(time.Second * 3))

	// should have executed
	want = []string{"task-1"}
	if !reflect.DeepEqual(tests.O.Get(), want) {
		t.Errorf("setup task execution order is not as expected, got: %v", tests.O.Get())
	}
}
