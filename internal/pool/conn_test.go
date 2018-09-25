// Copyright 2018 Changkun Ou. All rights reserved.
// Use of this source code is governed by a MIT
// license that can be found in the LICENSE file.

package pool_test

import (
	"testing"

	"github.com/changkun/sched/internal/pool"
)

func TestInit(t *testing.T) {
	p := pool.Init("redis://127.0.0.1/1")
	if p == nil {
		t.Errorf("pool initialization must not empty!")
	}
}

func TestGet(t *testing.T) {
	conn := pool.Get()
	if conn == nil {
		t.Errorf("pool.Get() must return non empty conn!")
	}
}

func BenchmarkGet(b *testing.B) {
	pool.Init("redis://127.0.0.1/1")
	for i := 0; i < b.N; i++ {
		pool.Get()
	}
}
