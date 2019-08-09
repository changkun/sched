// Copyright 2018 Changkun Ou. All rights reserved.
// Use of this source code is governed by a MIT
// license that can be found in the LICENSE file.

package sched

import (
	"time"

	"github.com/go-redis/redis"
)

type cache struct {
	client *redis.Client
}

func newCache(url string) (*cache, error) {
	opt, err := redis.ParseURL(url)
	if err != nil {
		return nil, err
	}
	return &cache{redis.NewClient(opt)}, nil
}

// GET command of redis
func (c *cache) Get(key string) (value string, err error) {
	return c.client.Get(key).Result()
}

// SET command of redis
func (c *cache) Set(key, value string) (err error) {
	return c.client.Set(key, value, 0).Err()
}

// DEL command of redis
func (c *cache) Del(key string) (err error) {
	return c.client.Del(key).Err()
}

// SETNX command of redis
func (c *cache) SetNX(key string, value string, expire time.Duration) (ok bool, err error) {
	return c.client.SetNX(key, value, expire).Result()
}

// KEYS command of redis
func (c *cache) Keys(prefix string) (keys []string, err error) {
	return c.client.Keys(prefix + "*").Result()
}

func (c *cache) Close() error {
	return c.client.Close()
}
