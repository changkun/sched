// Copyright 2018 Changkun Ou. All rights reserved.
// Use of this source code is governed by a MIT
// license that can be found in the LICENSE file.

package store

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/changkun/sched/task"
)

const (
	prefixTask = "sched:task:"
	prefixLock = "sched:lock:"
)

// Lock the given task
func Lock(t task.Interface) (bool, error) {
	return SETNX(prefixLock+t.GetID(), "locking", t.GetTimeout())
}

// Unlock the given task explicitly
func Unlock(t task.Interface) (err error) {
	return DEL(prefixLock + t.GetID())
}

// Save record into data store
func Save(t task.Interface) error {
	r := &Record{
		ID:        t.GetID(),
		Execution: t.GetExecution(),
		Data:      t,
	}
	return r.Save()
}

// Delete deletes record by id
func Delete(t task.Interface) error {
	return DEL(prefixTask + t.GetID())
}

// Record of a schedule
type Record struct {
	ID        string      `json:"id"`
	Execution time.Time   `json:"execution"`
	Data      interface{} `json:"data"`
}

// GetRecords all records keys
func GetRecords() (keys []string, err error) {
	keys, err = KEYS(prefixTask)
	ids := []string{}
	for _, key := range keys {
		ids = append(ids, strings.TrimPrefix(key, prefixTask))
	}
	return ids, err
}

// Read record with specified ID
func (r *Record) Read(id string) (err error) {
	reply, err := GET(prefixTask + id)
	return json.Unmarshal([]byte(reply), r)
}

// Save record into data store
func (r *Record) Save() (err error) {
	data, err := json.Marshal(r)
	return SET(prefixTask+r.ID, string(data))
}
