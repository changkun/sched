// Copyright 2018-2019 Changkun Ou. All rights reserved.
// Use of this source code is governed by a MIT
// license that can be found in the LICENSE file.

// +build !go1.13

package sched

import (
	"encoding/json"
	"reflect"
)

func (s *sched) load(id string, t Task) (TaskFuture, error) {
	r := &record{ID: id}
	if err := r.read(); err != nil {
		return nil, err
	}

	data, _ := json.Marshal(r.Data)

	// temp1 holds for a unset value of t, and temp2 tries to be set by json.Unmarshal.
	//
	// In the end, if temp1 and temp2 are appropriate type of tasks, then temp2 should
	// not DeepEqual to temp1, because temp2 is setted by store data.
	// Otherwise, the determined task is inappropriate type to be scheduled, jump to
	// next record and see if it can be scheduled.
	temp1 := reflect.New(reflect.ValueOf(t).Elem().Type()).Interface().(Task)
	temp2 := reflect.New(reflect.ValueOf(t).Elem().Type()).Interface().(Task)
	json.Unmarshal(data, &temp2)
	if reflect.DeepEqual(temp1, temp2) || temp2 == nil || !temp2.IsValidID() {
		return nil, nil
	}
	temp2.SetID(id)
	temp2.SetExecution(r.Execution)
	future, _ := s.tasks.push(temp2)
	return future, nil
}
