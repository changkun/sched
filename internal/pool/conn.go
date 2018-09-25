// Copyright 2018 Changkun Ou. All rights reserved.
// Use of this source code is governed by a MIT
// license that can be found in the LICENSE file.

package pool

import (
	"sync"
	"time"

	"github.com/gomodule/redigo/redis"
)

// Init connection pool of redis
func Init(url string) *redis.Pool {
	return newPool(url)
}

// Get connection from connection pool
func Get() redis.Conn {
	return pool.Get()
}

var (
	oncePool sync.Once
	pool     *redis.Pool
)

// newPool creates a redis connection pool
func newPool(url string) *redis.Pool {
	oncePool.Do(func() {
		pool = &redis.Pool{
			MaxIdle:     10,
			IdleTimeout: 240 * time.Second,
			Dial: func() (redis.Conn, error) {
				return redis.DialURL(url)
			},
		}
	})
	return pool
}
