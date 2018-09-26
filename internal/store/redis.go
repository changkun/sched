// Copyright 2018 Changkun Ou. All rights reserved.
// Use of this source code is governed by a MIT
// license that can be found in the LICENSE file.

package store

import (
	"time"

	"github.com/changkun/sched/internal/pool"
	"github.com/gomodule/redigo/redis"
)

// GET command of redis
func GET(key string) (value string, err error) {
	conn := pool.Get()
	defer conn.Close()
	value, err = redis.String(conn.Do("GET", key))
	return
}

// SET command of redis
func SET(key, value string) (err error) {
	conn := pool.Get()
	defer conn.Close()

	_, err = conn.Do("SET", key, value)
	return
}

// DEL command of redis
func DEL(key string) (err error) {
	conn := pool.Get()
	defer conn.Close()

	_, err = conn.Do("DEL", key)
	return
}

// SETNX command of redis
func SETNX(key string, value string, expire time.Duration) (ok bool, err error) {
	conn := pool.Get()
	defer conn.Close()
	reply, err := redis.String(conn.Do("SET", key, value, "EX", expire.Seconds(), "NX"))
	if reply != "OK" {
		ok = false
		return
	}
	ok = true
	return
}

// KEYS command of redis
func KEYS(prefix string) (keys []string, err error) {
	conn := pool.Get()
	defer conn.Close()
	keys, err = redis.Strings(conn.Do("KEYS", prefix+"*"))
	return
}
