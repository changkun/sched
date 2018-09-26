// Copyright 2018 Changkun Ou. All rights reserved.
// Use of this source code is governed by a MIT
// license that can be found in the LICENSE file.

package store_test

import (
	"reflect"
	"testing"
	"time"

	"github.com/changkun/sched/internal/pool"
	"github.com/changkun/sched/internal/store"
	"github.com/changkun/sched/tests"
)

func TestRedis(t *testing.T) {
	pool.Init("redis://127.0.0.1/1")

	if err := store.SET("123", "456"); err != nil {
		t.Fatal("SET must success!")
	}
	val, err := store.GET("123")
	if err != nil {
		t.Errorf("GET must success, err: %v", err)
	}
	keys, err := store.KEYS("")
	if err != nil {
		t.Errorf("KEYS must success, err: %v", err)
	}
	if !reflect.DeepEqual(keys, []string{"123"}) {
		t.Errorf("KEYS must success, got: %v", keys)
	}
	if val != "456" {
		t.Errorf("GET wrong, want 456, got %s", val)
	}
	if err := store.DEL("123"); err != nil {
		t.Fatal("DEL must success")
	}
	_, err = store.GET("123")
	if err == nil {
		t.Errorf("GET empty must fail, got nil")
	}
	if _, err := store.SETNX("789", "321", time.Second); err != nil {
		t.Fatal("SETEX must success!")
	}
	time.Sleep(time.Second * 2)
	_, err = store.GET("789")
	if err == nil {
		t.Errorf("GET empty must fail, got nil")
	}
}

func TestSETEXFail(t *testing.T) {
	pool.Init("redis://127.0.0.1/1")

	ok, _ := store.SETNX("111", "222", -time.Second)
	if ok {
		t.Errorf("setex should not be ok!")
	}
}

func TestStoreLockUnlock(t *testing.T) {
	pool.Init("redis://127.0.0.1/8")
	task := tests.NewTask("task-0", time.Now().UTC().Add(time.Second))

	// Lock first
	ok, err := store.Lock(task)
	if err != nil {
		t.Fatal("lock task fail: ", err)
	}
	if !ok {
		t.Fatal("lock task fail: not ok")
	}

	// Lock again
	_, err = store.Lock(task)
	if err == nil {
		t.Fatal("lock task return non nil: ", err)
	}

	// unlock
	if err := store.Unlock(task); err != nil {
		t.Errorf("DEL after lock fail: %v", err)
	}
}

func TestStoreSaveDelete(t *testing.T) {
	pool.Init("redis://127.0.0.1/8")
	task := tests.NewTask("task-0", time.Now().UTC().Add(time.Second))

	err := store.Save(task)
	if err != nil {
		t.Errorf("store save record fail: %v", err)
	}

	keys, err := store.GetRecords()
	if err != nil {
		t.Errorf("store get records fail: %v", err)
	}
	if !reflect.DeepEqual(keys, []string{"task-0"}) {
		t.Errorf("store get records fail, got %v", keys)
	}

	r := &store.Record{}
	if err := r.Read("task-0"); err != nil {
		t.Errorf("read task fail: %v", err)
	}

	err = store.Delete(task)
	if err != nil {
		t.Errorf("store delete record fail: %v", err)
	}
}
