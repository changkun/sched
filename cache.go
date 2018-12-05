// Copyright 2018 Changkun Ou. All rights reserved.
// Use of this source code is governed by a MIT
// license that can be found in the LICENSE file.

package sched

import (
	"time"

	"github.com/gomodule/redigo/redis"
)

var cache0 *cache

func connectCache(url string) {
	cache0 = &cache{
		pool: &redis.Pool{
			MaxIdle:     200,
			IdleTimeout: 240 * time.Second,
			Dial: func() (redis.Conn, error) {
				return redis.DialURL(url)
			},
		},
	}
}

type cache struct {
	pool *redis.Pool
}

// GET command of redis
func (c *cache) GET(key string) (value string, err error) {
	conn := c.pool.Get()
	defer conn.Close()
	value, err = redis.String(conn.Do("GET", key))
	return
}

// SET command of redis
func (c *cache) SET(key, value string) (err error) {
	conn := c.pool.Get()
	defer conn.Close()

	_, err = conn.Do("SET", key, value)
	return
}

// DEL command of redis
func (c *cache) DEL(key string) (err error) {
	conn := c.pool.Get()
	defer conn.Close()

	_, err = conn.Do("DEL", key)
	return
}

// SETNX command of redis
func (c *cache) SETNX(key string, value string, expire time.Duration) (ok bool, err error) {
	conn := c.pool.Get()
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
func (c *cache) KEYS(prefix string) (keys []string, err error) {
	conn := c.pool.Get()
	defer conn.Close()
	keys, err = redis.Strings(conn.Do("KEYS", prefix+"*"))
	return
}

// EXISTS command of redis
func (c *cache) EXISTS(key string) (ok bool, err error) {
	conn := c.pool.Get()
	defer conn.Close()

	ok, err = redis.Bool(conn.Do("EXISTS", key))
	return
}

func (c *cache) TTL(key string) (time.Duration, error) {
	conn := c.pool.Get()
	defer conn.Close()

	n, err := redis.Int64(conn.Do("TTL", key))
	return time.Duration(n), err
}

func (c *cache) PERSIST(key string) (ok bool, err error) {
	conn := c.pool.Get()
	defer conn.Close()

	ok, err = redis.Bool(conn.Do("PERSIST", key))
	return
}

func (c *cache) Close() {
	c.pool.Close()
}
