// Copyright 2018-2019 Changkun Ou. All rights reserved.
// Use of this source code is governed by a MIT
// license that can be found in the LICENSE file.

// +build go1.13

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

	v := reflect.New(reflect.ValueOf(t).Elem().Type()).Interface().(Task)
	json.Unmarshal(data, &v)
	if v == nil || reflect.ValueOf(v).Elem().IsZero() || !v.IsValidID() {
		return nil, nil
	}
	v.SetID(id)
	v.SetExecution(r.Execution)
	future, _ := s.tasks.push(v)
	return future, nil
}
