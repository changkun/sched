package pool_test

import (
	"testing"

	"github.com/changkun/goscheduler/internal/pool"
	"github.com/stretchr/testify/assert"
)

func TestInit(t *testing.T) {
	assert.NotNil(t, pool.Init("redis://127.0.0.1/1"), "pool must not empty!")
}

func TestGet(t *testing.T) {
	assert.NotNil(t, pool.Get(), "redis.Conn must not nil")
}
