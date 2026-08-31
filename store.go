// Copyright 2018 Changkun Ou. All rights reserved.
// Use of this source code is governed by a MIT
// license that can be found in the LICENSE file.

package sched

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

// Key prefixes in the store. Tasks live under prefixTask and survive a
// restart; locks live under prefixLock and expire with Task.Timeout.
const (
	prefixTask = "sched:task:"
	prefixLock = "sched:lock:"
)

// store is the persistence sched needs: a string key-value space with an
// atomic set-if-absent and a key scan. Redis provides it; the interface
// exists so the scheduler holds no Redis type and so tests can inject
// failures.
type store interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key, value string) error
	Del(ctx context.Context, key string) error
	SetNX(ctx context.Context, key, value string, ttl time.Duration) (bool, error)
	Keys(ctx context.Context, prefix string) ([]string, error)
	Close() error
}

// redisStore implements store on a Redis server.
type redisStore struct {
	client *redis.Client
}

func newRedisStore(url string) (*redisStore, error) {
	opt, err := redis.ParseURL(url)
	if err != nil {
		return nil, err
	}
	return &redisStore{client: redis.NewClient(opt)}, nil
}

func (s *redisStore) Get(ctx context.Context, key string) (string, error) {
	return s.client.Get(ctx, key).Result()
}

func (s *redisStore) Set(ctx context.Context, key, value string) error {
	return s.client.Set(ctx, key, value, 0).Err()
}

func (s *redisStore) Del(ctx context.Context, key string) error {
	return s.client.Del(ctx, key).Err()
}

func (s *redisStore) SetNX(ctx context.Context, key, value string, ttl time.Duration) (bool, error) {
	return s.client.SetNX(ctx, key, value, ttl).Result()
}

func (s *redisStore) Keys(ctx context.Context, prefix string) ([]string, error) {
	return s.client.Keys(ctx, prefix+"*").Result()
}

func (s *redisStore) Close() error { return s.client.Close() }

// record is the persisted form of a scheduled task.
type record struct {
	ID        string    `json:"id"`
	Execution time.Time `json:"execution"`
	Data      any       `json:"data"`
}

// saveTask writes the task so that it can be recovered after a restart.
func saveTask(ctx context.Context, s store, t Task) error {
	data, err := json.Marshal(&record{
		ID:        t.ID(),
		Execution: t.Execution(),
		Data:      t,
	})
	if err != nil {
		return err
	}
	return s.Set(ctx, prefixTask+t.ID(), string(data))
}

// readRecord reads the persisted record of the task with the given id.
func readRecord(ctx context.Context, s store, id string) (*record, error) {
	reply, err := s.Get(ctx, prefixTask+id)
	if err != nil {
		return nil, err
	}
	r := &record{ID: id}
	if err := json.Unmarshal([]byte(reply), r); err != nil {
		return nil, err
	}
	return r, nil
}

// taskIDs lists the identifiers of every persisted task.
func taskIDs(ctx context.Context, s store) ([]string, error) {
	keys, err := s.Keys(ctx, prefixTask)
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(keys))
	for _, key := range keys {
		ids = append(ids, strings.TrimPrefix(key, prefixTask))
	}
	return ids, nil
}
