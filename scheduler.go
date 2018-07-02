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
	Execute() error
	// FailRetryDuration returns the task retry duration if fails
	FailRetryDuration() time.Duration
}

// Config provides the database URI for goschduler
type Config struct {
	DatabaseURI string
}

// Init creates the connection of database
func Init(config *Config) error {
	s := getScheduler()
	if s.db == nil {
		option, err := redis.ParseURL(config.DatabaseURI)
		if err != nil {
			return err
		}
		s.db = redis.NewClient(option)
	}
	return nil
}

// Poll recover tasks from database when application boot up
func Poll(t Task) error {
	s := getScheduler()
	err := s.initTasks(t)
	if err != nil {
		return err
	}
	return nil
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
	if runner.Timer != nil {
		// timer.Stop can fail in the following case:
		//  - a timer has executed: this is not possible in our implementation because timer will be nil after execution
		//  - user created a timer directly, without invoking startTimer: not our case
		// see https://github.com/golang/go/blob/d6c3b0a56d8c81c221b0adf69ae351f7cd467854/src/runtime/time.go#L177
		runner.Timer.Stop()
	}
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
var retryNum int
var retryDuration time.Duration

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
func (s scheduler) initTasks(t Task) error {
	keys, err := s.db.Keys(prefix + "*").Result()
	if err != nil {
		return err
	}
	for _, key := range keys {
		s.recoverTask(t, key)
	}
	return nil
}
func (s scheduler) recoverTask(t Task, key string) error {
	result, err := s.db.Get(key).Result()
	if err != nil {
		return err
	}
	r := &record{}
	if err := json.Unmarshal([]byte(result), r); err != nil {
		return err
	}
	// the following json (Un)Marshal is not possible return err if r is unmarshaled success
	bytes, _ := json.Marshal(r.Data)
	temp := reflect.New(reflect.ValueOf(t).Elem().Type()).Interface().(Task)
	json.Unmarshal(bytes, &temp)

	return Schedule(temp)
}
func (s scheduler) reschedule(t Task) {
	s.getRunnerBy(t.Identifier()).Timer = time.NewTimer(
		time.Duration(t.GetExecuteTime().Sub(time.Now().UTC())),
	)
	go func(t Task) {
		<-s.getRunnerBy(t.Identifier()).Timer.C
		s.getRunnerBy(t.Identifier()).Timer = nil

		if t.GetExecuteTime().Before(time.Now().UTC()) {
			go func(t Task) {
				if err := t.Execute(); err != nil {
					// failed retry
					t.SetExecuteTime(t.GetExecuteTime().Add(t.FailRetryDuration()))
					Schedule(t)
					return
				}
				s.db.Del(prefix + t.Identifier()).Result()
				delete(*s.getManager(), t.Identifier())
			}(t)
			return
		}
		// no need to stop timer since it already expired
		s.reschedule(t)
	}(t)
}
