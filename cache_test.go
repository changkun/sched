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

	t.Run("cache-connect", func(t *testing.T) {
		connectCache("redis://127.0.0.1:6379/2")
		err := cache0.pool.Close()
		if err != nil {
			t.Fatalf("close cache pool error: %v", err)
		}
	})

	connectCache("redis://127.0.0.1:6379/2")

	t.Run("cache-set", func(t *testing.T) {
		err := cache0.SET(key, val)
		if err != nil {
			t.Fatal("cache SET must success!")
		}
	})

	t.Run("cache-get", func(t *testing.T) {
		v, err := cache0.GET(key)
		if err != nil {
			t.Fatalf("cache GET must success, err: %v", err)
		}
		if v != val {
			t.Fatalf("cache GET wrong, want %s, got %s", val, v)
		}
	})

	t.Run("cache-keys", func(t *testing.T) {
		keys, err := cache0.KEYS(key)
		if err != nil {
			t.Fatalf("cache KEYS must success, err: %v", err)
		}
		if !reflect.DeepEqual(keys, []string{key}) {
			t.Fatalf("cache KEYS must success, got: %v", keys)
		}
	})

	t.Run("cache-del", func(t *testing.T) {
		err := cache0.DEL(key)
		if err != nil {
			t.Fatal("cache DEL must success")
		}
		_, err = cache0.GET(key)
		if err == nil {
			t.Fatal("cache GET after DEL must error, got nil")
		}
	})

	t.Run("cache-setnx-success", func(t *testing.T) {
		_, err := cache0.SETNX(key, val, time.Second)
		if err != nil {
			t.Fatal("SETEX must success!")
		}
		time.Sleep(time.Second * 2)
		_, err = cache0.GET(key)
		if err == nil {
			t.Errorf("cache GET empty must error, got nil")
		}
	})

	t.Run("cache-setnx-fail", func(t *testing.T) {
		_, err := cache0.SETNX(key, val, time.Second)
		if err != nil {
			t.Fatal("cache SETNX must success!")
		}
		_, err = cache0.SETNX(key, val, time.Second)
		if err == nil {
			t.Fatal("cache SETNX must fail after SETNX!")
		}
		time.Sleep(time.Second * 2)
		_, err = cache0.GET(key)
		if err == nil {
			t.Errorf("cache GET empty must error, got nil")
		}
	})

	t.Run("cache-exists-ttl-persist", func(t *testing.T) {
		_, err := cache0.SETNX(key, val, time.Second)
		if err != nil {
			t.Fatal("cache SETNX must success!")
		}
		ok, err := cache0.EXISTS(key)
		if err != nil {
			t.Fatalf("cache EXISTS must success, err: %v", err)
		}
		if !ok {
			t.Fatal("cache EXISTS must return exist, got false")
		}
		d, err := cache0.TTL(key)
		if err != nil {
			t.Fatalf("cache TTL must success, err: %v", err)
		}
		if d < 0 {
			t.Fatalf("cache TTL get negative duration: %d", d)
		}
		ok, err = cache0.PERSIST(key)
		if err != nil {
			t.Fatalf("cache PERSIST must success, err: %v", err)
		}
		if !ok {
			t.Fatal("cache PERSIST must return true, got false")
		}
		err = cache0.DEL(key)
		if err != nil {
			t.Fatal("cache DEL must success")
		}
	})

	// err := cache0.pool.Close()
	// if err != nil {
	// 	t.Fatalf("close cache pool error: %v", err)
	// }
}
