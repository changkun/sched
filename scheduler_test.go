package goscheduler

import (
	"fmt"
	"testing"
	"time"
)

func TestInitializer(t *testing.T) {
	Initialize(&DatabaseConfig{
		URI: "",
	})
	Initialize(&DatabaseConfig{
		URI: "redis://127.0.0.1:6379/8",
	})
}

type CustomTask struct {
	ID          string    `json:"uuid"`
	Start       time.Time `json:"start"`
	End         time.Time `json:"end"`
	Information string    `json:"info"`
}

func (c CustomTask) Identifier() string {
	return c.ID
}
func (c *CustomTask) SetExecuteTime(t time.Time) time.Time {
	c.End = t
	return c.End
}
func (c CustomTask) GetExecuteTime() time.Time {
	return c.End
}
func (c CustomTask) Execute() {
	fmt.Println("Custom Task is Running: ", c.Information)
}

func TestPoller(t *testing.T) {
	Initialize(&DatabaseConfig{
		URI: "redis://127.0.0.1:6379/8",
	})
	s := getScheduler()
	task1 := &CustomTask{
		ID:          "123",
		Start:       time.Now().UTC(),
		End:         time.Now().UTC().Add(time.Duration(1) * time.Second),
		Information: "TestSchedule message 1",
	}
	task2 := &CustomTask{
		ID:          "456",
		Start:       time.Now().UTC(),
		End:         time.Now().UTC().Add(time.Duration(2) * time.Second),
		Information: "TestSchedule message 2",
	}
	s.save(task1)
	s.save(task2)

	var customType CustomTask
	Poller(&customType)
	time.Sleep(time.Second * time.Duration(5))
}

func TestSchedule(t *testing.T) {
	task1 := &CustomTask{
		ID:          "123",
		Start:       time.Now().UTC(),
		End:         time.Now().UTC().Add(time.Duration(1) * time.Second),
		Information: "TestSchedule message 1",
	}
	task2 := &CustomTask{
		ID:          "456",
		Start:       time.Now().UTC(),
		End:         time.Now().UTC().Add(time.Duration(1) * time.Second),
		Information: "TestSchedule message 2",
	}
	Initialize(&DatabaseConfig{
		URI: "redis://127.0.0.1:6379/8",
	})
	if err := Schedule(task1); err != nil {
		t.Errorf("error: %s", err.Error())
	}
	if err := Schedule(task2); err != nil {
		t.Errorf("error: %s", err.Error())
	}
	task1.SetExecuteTime(task1.GetExecuteTime().Add(time.Second))
	if err := Schedule(task1); err != nil {
		t.Errorf("error: %s", err.Error())
	}
	task1.SetExecuteTime(task1.GetExecuteTime().Add(time.Second))
	s := getScheduler()
	s.save(task1)
	time.Sleep(time.Second * time.Duration(3))
}

func TestBoot(t *testing.T) {
	Initialize(&DatabaseConfig{
		URI: "redis://127.0.0.1:6379/8",
	})

	task := &CustomTask{
		ID:          "456",
		Start:       time.Now().UTC(),
		End:         time.Now().UTC().Add(time.Duration(1) * time.Second),
		Information: "TestBoot message",
	}
	task.SetExecuteTime(task.GetExecuteTime().Add(time.Second * 10))
	Schedule(task)

	s := getScheduler()
	// will fail
	Boot(&CustomTask{
		ID:          "123",
		Start:       time.Now().UTC(),
		End:         time.Now().UTC().Add(time.Duration(1) * time.Second),
		Information: "TestBoot message",
	})
	s.db.Del(prefix + "123")

	// will fail
	s.db.Set(prefix+"777", "123123123", 0).Result()
	Boot(&CustomTask{
		ID:          "777",
		Start:       time.Now().UTC(),
		End:         time.Now().UTC().Add(time.Duration(1) * time.Second),
		Information: "TestBoot message",
	})
	s.db.Del(prefix + "777")

	// will success
	Boot(task)
	time.Sleep(time.Second)
}

type Func func()

func (c Func) Identifier() string {
	return ""
}
func (c *Func) SetExecuteTime(t time.Time) time.Time {
	return time.Now()
}
func (c Func) GetExecuteTime() time.Time {
	return time.Now()
}
func (c Func) Execute() {
	return
}

func TestPollerFail(t *testing.T) {
	Initialize(&DatabaseConfig{
		URI: "redis://127.0.0.1:6379/8",
	})
	s := getScheduler()
	s.db.Set(prefix+"777", "123123123", 0).Result()
	var c CustomTask
	if err := s.initTasks(&c); err != nil {
		s.db.Del(prefix + "777")
	}
}

func TestSaveFail(t *testing.T) {
	f := Func(func() { return })
	s := getScheduler()
	s.save(&f)
}
