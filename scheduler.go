package goscheduler

import (
	"encoding/json"
	"reflect"
	"sync"
	"time"

	"github.com/go-redis/redis"
)

// Task interface defines the task need to be executed
type Task interface {
	// Identifier should returns a unique string for the task, usually can return an UUID
	Identifier() string
	// GetExecuteTime should returns the excute time of the task
	GetExecuteTime() time.Time
	// SetExecuteTime can set the execution time of the task
	SetExecuteTime(t time.Time) time.Time
	// Execute defines the actual running task
	Execute()
}

// DatabaseConfig provides the database URI for goschduler
type DatabaseConfig struct {
	URI string
}

// Init creates the connection of database
func Init(config *DatabaseConfig) error {
	s := getScheduler()
	if s.db == nil {
		option, err := redis.ParseURL(config.URI)
		if err != nil {
			return err
		}
		s.db = redis.NewClient(option)
	}
	return nil
}

// Poll restore tasks from database when application boot up
func Poll(t Task) {
	s := getScheduler()
	err := s.initTasks(t)
	if err != nil {
		panic(err)
	}
}

// Schedule a task
func Schedule(t Task) error {
	defer func() {
		recover()
	}()
	s := getScheduler()
	if err := s.save(t); err != nil {
		return err
	}
	runner := s.getRunnerBy(t.Identifier())

	// if no entries then start a new one
	if runner.Timer != nil {
		if stop := runner.Timer.Stop(); !stop {
			// Task already start
			return nil
		}
	}

	// otherwise reschedule the task
	s.reschedule(t)
	return nil
}

// Boot a task immediately
func Boot(t Task) error {
	t.SetExecuteTime(time.Now().UTC())
	Schedule(t)
	return nil
}

const prefix = "goscheduler:"

var onceManager sync.Once
var onceSchduler sync.Once
var worker *scheduler

type taskManager map[string]*taskRunner
type taskRunner struct {
	UUID  string
	Timer *time.Timer
}
type scheduler struct {
	db      *redis.Client
	manager *taskManager
}
type record struct {
	Identifier string      `json:"identifier"`
	Execution  time.Time   `json:"execution"`
	Data       interface{} `json:"data"`
}

func getScheduler() *scheduler {
	onceSchduler.Do(func() {
		worker = &scheduler{}
	})
	return worker
}
func (s *scheduler) getManager() *taskManager {
	onceManager.Do(func() {
		s.manager = &taskManager{}
	})
	return s.manager
}
func (s *scheduler) getRunnerBy(uuid string) *taskRunner {
	if runner, ok := (*s.getManager())[uuid]; ok {
		return runner
	}
	runner := &taskRunner{
		UUID:  uuid,
		Timer: nil,
	}
	(*s.getManager())[uuid] = runner
	return runner
}
func (s *scheduler) save(t Task) error {
	r := record{
		Identifier: t.Identifier(),
		Execution:  t.GetExecuteTime(),
		Data:       t,
	}
	bytes, err := json.Marshal(&r)
	if err != nil {
		return err
	}
	if _, err := s.db.Set(
		prefix+t.Identifier(),
		string(bytes),
		0,
	).Result(); err != nil {
		return err
	}
	return nil
}
func (s scheduler) initTasks(task Task) error {
	keys, err := s.db.Keys(prefix + "*").Result()
	if err != nil {
		return err
	}
	for _, key := range keys {
		result, err := s.db.Get(key).Result()
		if err != nil {
			return err
		}
		r := &record{}
		if err := json.Unmarshal([]byte(result), r); err != nil {
			return err
		}
		// the following json (Un)Marshal is not possible if r is unmarshaled success
		bytes, _ := json.Marshal(r.Data)
		temp := reflect.New(reflect.ValueOf(task).Elem().Type()).Interface().(Task)
		json.Unmarshal(bytes, &temp)

		if err := Schedule(temp); err != nil {
			return err
		}
	}
	return nil
}
func (s scheduler) reschedule(t Task) {
	defer func() {
		recover()
	}()
	s.getRunnerBy(t.Identifier()).Timer = time.NewTimer(
		time.Duration(t.GetExecuteTime().Sub(time.Now().UTC())),
	)
	go func(t Task) {
		<-s.getRunnerBy(t.Identifier()).Timer.C
		s.execute(t)
	}(t)
}
func (s scheduler) execute(t Task) {
	defer func() {
		recover()
	}()
	if t.GetExecuteTime().Before(time.Now().UTC()) {
		go func(t Task) {
			t.Execute()
			s.db.Del(prefix + t.Identifier()).Result()
			delete(*s.getManager(), t.Identifier())
		}(t)
		return
	}
	s.getRunnerBy(t.Identifier()).Timer.Stop()
	s.reschedule(t)
	return
}
