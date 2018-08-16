package goscheduler

import (
	"encoding/json"
	"reflect"
	"sync"
	"time"

	"github.com/go-redis/redis"
)

const (
	prefixTask = "goscheduler:task:"
	prefixLock = "goscheduler:lock:"
)

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

func newScheduler() *scheduler {
	onceSchduler.Do(func() {
		worker = &scheduler{
			db:      nil,
			manager: new(sync.Map),
		}
	})
	return worker
}
func (s *scheduler) initDB(url string) error {
	if s.db == nil {
		option, err := redis.ParseURL(url)
		if err != nil {
			return err
		}
		s.db = redis.NewClient(option)
	}
	return nil
}
func (s *scheduler) recover(t Task) error {
	keys, err := newScheduler().db.Keys(prefixTask + "*").Result()
	if err != nil {
		return err
	}
	for _, key := range keys {
		if err := recover(t, key); err != nil {
			return err
		}
	}
	return nil
}
func (s *scheduler) getRunner(t Task) *time.Timer {
	if v, ok := s.manager.Load(t.Identifier()); v != nil && ok {
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
	if _, err := s.db.Set(prefixTask+t.Identifier(), string(bytes), 0).Result(); err != nil {
		return err
	}
	return nil
}
func (s *scheduler) store(t Task, v interface{}) {
	s.manager.Store(t.Identifier(), v)
}
func (s *scheduler) delete(t Task) {
	s.db.Del(prefixTask + t.Identifier()).Result()
	s.manager.Delete(t.Identifier())
}
func (s *scheduler) lock(t Task, expiration time.Duration) (bool, error) {
	bytes, err := json.Marshal(&record{
		Identifier: t.Identifier(),
		Execution:  t.GetExecuteTime(),
		Data:       t,
	})
	if err != nil {
		return false, err
	}
	return s.db.SetNX(prefixLock+t.Identifier(), string(bytes), expiration).Result()
}
func (s *scheduler) unlock(t Task) error {
	if _, err := s.db.Del(prefixLock + t.Identifier()).Result(); err != nil {
		return err
	}
	return nil
}
func (s *scheduler) getExecuteTime(t Task) (*time.Time, error) {
	result, err := s.db.Get(prefixTask + t.Identifier()).Result()
	if err != nil {
		return nil, err
	}
	r := &record{}
	// this is not possible to fail
	json.Unmarshal([]byte(result), r)
	return &r.Execution, nil
}

func recover(t Task, key string) error {
	result, err := newScheduler().db.Get(key).Result()
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
	newScheduler().store(t, timer)
	go func(t Task) {
		<-timer.C
		executeOnce(t)
	}(t)
}
func executeOnce(t Task) {
	s := newScheduler()
	// lock the task execution for one scheduler instance
	// cancel execution if lock error or already locked
	if ok, err := s.lock(t, time.Second*10); err != nil || !ok {
		return
	}
	// unlock the task after execute, no matter
	// it excutes success, need reschedule, or should retry
	defer s.unlock(t)

	execute(t)
}

func execute(t Task) {
	s := newScheduler()
	s.store(t, nil)
	// final check before execution, from database
	execution, err := s.getExecuteTime(t)
	if err != nil {
		return
	}
	if execution.Before(time.Now().UTC()) {
		if err := t.Execute(); err != nil {
			// no need to unlock the task if failed
			retry(t)
			return
		}
		s.delete(t)
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
