package store

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/changkun/goscheduler/task"
)

const (
	prefixTask = "goscheduler:task:"
	prefixLock = "goscheduler:lock:"
)

// Lock the given task
func Lock(t task.Interface) (bool, error) {
	return SETEX(prefixLock+t.GetID(), "locking", t.GetTimeout())
}

// Unlock the given task explicitly
func Unlock(t task.Interface) (err error) {
	return DEL(prefixLock + t.GetID())
}

// Record of a schedule
type Record struct {
	ID        string      `json:"id"`
	Execution time.Time   `json:"execution"`
	Data      interface{} `json:"data"`
}

// Save record into data store
func Save(t task.Interface) error {
	r := &Record{
		ID:        t.GetID(),
		Execution: t.GetExecution(),
		Data:      t,
	}
	if err := r.Save(); err != nil {
		return err
	}
	return nil
}

// Delete deletes record by id
func Delete(t task.Interface) error {
	return DEL(prefixTask + t.GetID())
}

// GetRecords all records keys
func GetRecords() ([]string, error) {
	keys, err := KEYS(prefixTask)
	if err != nil {
		return nil, err
	}
	ids := []string{}
	for _, key := range keys {
		ids = append(ids, strings.TrimPrefix(key, prefixTask))
	}
	return ids, nil
}

// Read record with specified ID
func (r *Record) Read(id string) error {
	reply, err := GET(prefixTask + id)
	if err != nil {
		return err
	}
	return json.Unmarshal([]byte(reply), r)
}

// Save record into data store
func (r *Record) Save() error {
	data, err := json.Marshal(r)
	if err != nil {
		return err
	}
	return SET(prefixTask+r.ID, string(data))
}
