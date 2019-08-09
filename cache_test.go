// Copyright 2018 Changkun Ou. All rights reserved.
// Use of this source code is governed by a MIT
// license that can be found in the LICENSE file.

package sched

import (
	"reflect"
	"testing"
	"time"
)

func TestRedis(t *testing.T) {
	key := "123"
	val := "456"
	url := "redis://127.0.0.1:6379/2"

	t.Run("cache-connect", func(t *testing.T) {
		c, err := newCache(url)
		if err != nil {
			t.Fatalf("new client failed: %v", err)
		}
		err = c.Close()
		if err != nil {
			t.Fatalf("close cache pool error: %v", err)
		}
	})

	c, err := newCache(url)
	if err != nil {
		t.Fatalf("new client failed: %v", err)
	}

	t.Run("cache-set", func(t *testing.T) {
		err := c.Set(key, val)
		if err != nil {
			t.Fatal("cache SET must success!")
		}
	})

	t.Run("cache-get", func(t *testing.T) {
		v, err := c.client.Get(key).Result()
		if err != nil {
			t.Fatalf("cache GET must success, err: %v", err)
		}
		if v != val {
			t.Fatalf("cache GET wrong, want %s, got %s", val, v)
		}
	})

	t.Run("cache-keys", func(t *testing.T) {
		keys, err := c.Keys(key)
		if err != nil {
			t.Fatalf("cache KEYS must success, err: %v", err)
		}
		if !reflect.DeepEqual(keys, []string{key}) {
			t.Fatalf("cache KEYS must success, got: %v", keys)
		}
	})

	t.Run("cache-del", func(t *testing.T) {
		err := c.Del(key)
		if err != nil {
			t.Fatal("cache DEL must success")
		}
		_, err = c.Get(key)
		if err == nil {
			t.Fatal("cache GET after DEL must error, got nil")
		}
	})

	t.Run("cache-setnx-success", func(t *testing.T) {
		_, err := c.SetNX(key, val, time.Second)
		if err != nil {
			t.Fatal("SETEX must success!")
		}
		time.Sleep(time.Second * 2)
		_, err = c.Get(key)
		if err == nil {
			t.Errorf("cache GET empty must error, got nil")
		}
	})

	t.Run("cache-setnx-fail", func(t *testing.T) {
		_, err := c.SetNX(key, val, time.Second)
		if err != nil {
			t.Fatal("cache SETNX must success!")
		}
		ok, err := c.SetNX(key, "random", time.Second)
		if ok != false && err == nil {
			t.Fatal("cache SETNX must fail after SETNX!")
		}
		time.Sleep(time.Second * 2)
		_, err = c.Get(key)
		if err == nil {
			t.Errorf("cache GET empty must error, got nil")
		}
	})
	c.Close()
}
