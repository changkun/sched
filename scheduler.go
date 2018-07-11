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
	keys, err := getScheduler().db.Keys(prefix + "*").Result()
	if err != nil {
		return err
	}
	for _, key := range keys {
		if err := recoverTask(t, key); err != nil {
			return err
		}
	}
	return nil
}

// Schedule a task
func Schedule(t Task) error {
	s := getScheduler()
	if err := s.save(t); err != nil {
		return err
	}

	runner := getRunnerBy(t.Identifier())
	if runner != nil {
		// timer Stop() can fail in the following case:
		//  - a timer has executed: this is not possible in our implementation because timer will be nil after arrival
		//  - user created a timer directly, without invoking startTimer: not our case
		// see https://github.com/golang/go/blob/b5375d70d1d626595ec68568b68cc28dddc859d1/src/runtime/time.go#L166
		runner.Stop()
	}
	reschedule(t)
	return nil
}

// Boot a task immediately
func Boot(t Task) error {
	t.SetExecuteTime(time.Now().UTC())
	return Schedule(t)
}

const prefix = "goscheduler:"

var onceSchduler sync.Once
var worker *scheduler

type scheduler struct {
	db      *redis.Client
	manager *sync.Map
}
type record struct {
	Identifier string      `json:"identifier"`
	Execution  time.Time   `json:"execution"`
	Data       interface{} `json:"data"`
}

func getScheduler() *scheduler {
	onceSchduler.Do(func() {
		worker = &scheduler{
			db:      nil,
			manager: new(sync.Map),
		}
	})
	return worker
}

func getRunnerBy(uuid string) *time.Timer {
	if v, ok := getScheduler().manager.Load(uuid); v != nil && ok {
		return v.(*time.Timer)
	}
	return nil
}
func (s *scheduler) save(t Task) error {
	bytes, err := json.Marshal(&record{
		Identifier: t.Identifier(),
		Execution:  t.GetExecuteTime(),
		Data:       t,
	})
	if err != nil {
		return err
	}
	if _, err := s.db.Set(prefix+t.Identifier(), string(bytes), 0).Result(); err != nil {
		return err
	}
	return nil
}
func getExecuteTimeBy(uuid string) (*time.Time, error) {
	result, err := getScheduler().db.Get(prefix + uuid).Result()
	if err != nil {
		return nil, err
	}
	r := &record{}
	// this is not possible to fail
	json.Unmarshal([]byte(result), r)
	return &r.Execution, nil
}
func recoverTask(t Task, key string) error {
	result, err := getScheduler().db.Get(key).Result()
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
func reschedule(t Task) {
	timer := time.NewTimer(
		time.Duration(t.GetExecuteTime().Sub(time.Now().UTC())),
	)
	getScheduler().manager.Store(t.Identifier(), timer)
	go func(t Task) {
		<-timer.C
		getScheduler().manager.Store(t.Identifier(), nil)
		execute(t)
	}(t)
}
func execute(t Task) {
	execution, err := getExecuteTimeBy(t.Identifier())
	if err != nil {
		return
	}
	if execution.Before(time.Now().UTC()) {
		if err := t.Execute(); err != nil {
			retry(t)
			return
		}
		getScheduler().db.Del(prefix + t.Identifier()).Result()
		getScheduler().manager.Delete(t.Identifier())
		return
	}
	// no need to stop timer since it already expired
	reschedule(t)
}
func retry(t Task) {
	t.SetExecuteTime(t.GetExecuteTime().Add(t.FailRetryDuration()))
	if err := Schedule(t); err != nil {
		reschedule(t)
	}
}
