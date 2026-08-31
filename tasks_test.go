// Copyright 2018 Changkun Ou. All rights reserved.
// Use of this source code is governed by a MIT
// license that can be found in the LICENSE file.

package sched

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

// order records the identifiers of the tasks that ran, in order.
type order struct {
	mu    sync.Mutex
	seen  []string
	first time.Time
	last  time.Time
}

var o = &order{}

func (o *order) push(id string) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.seen = append(o.seen, id)
}

func (o *order) get() []string {
	o.mu.Lock()
	defer o.mu.Unlock()
	return append([]string(nil), o.seen...)
}

func (o *order) clear() {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.seen = nil
	o.first = time.Time{}
	o.last = time.Time{}
}

func (o *order) markFirst() {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.first.IsZero() {
		o.first = time.Now().UTC()
	}
}

func (o *order) markLast() {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.last = time.Now().UTC()
}

func (o *order) span() time.Duration {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.last.Sub(o.first)
}

// Base holds the state every test task shares. It is embedded by pointer so
// that the atomic execution time is never copied, and Public gives
// json.Marshal a field to write, without which a task cannot be recovered.
type Base struct {
	Public string `json:"public"`

	id   string
	exec atomic.Int64 // execution time in Unix nanoseconds
}

func newBase(id string, e time.Time) *Base {
	b := &Base{Public: "not nil", id: id}
	b.exec.Store(e.UnixNano())
	return b
}

func (t *Base) ID() string      { return t.id }
func (t *Base) SetID(id string) { t.id = id }

func (t *Base) Execution() time.Time {
	return time.Unix(0, t.exec.Load()).UTC()
}

func (t *Base) SetExecution(e time.Time) { t.exec.Store(e.UnixNano()) }

func (t *Base) Timeout() time.Duration { return time.Second }

func (t *Base) RetryTime() time.Time { return time.Now().UTC().Add(time.Second) }

// task is the ordinary task: it runs once and succeeds.
type task struct{ *Base }

func newTask(id string, e time.Time) *task { return &task{Base: newBase(id, e)} }

func (t *task) Execute() (any, bool, error) {
	o.push(t.id)
	return fmt.Sprintf("execute task %s.", t.id), false, nil
}

// nilTask returns no result, so sched substitutes a placeholder.
type nilTask struct{ *Base }

func newNilTask(id string, e time.Time) *nilTask { return &nilTask{Base: newBase(id, e)} }

func (t *nilTask) Execute() (any, bool, error) {
	o.push(t.id)
	return nil, false, nil
}

// panicTask panics, so sched must publish the panic through the Future.
type panicTask struct{ *Base }

func newPanicTask(id string, e time.Time) *panicTask { return &panicTask{Base: newBase(id, e)} }

func (t *panicTask) Execute() (any, bool, error) {
	o.push(t.id)
	panic(t.id)
}

// retryTask asks for a retry until it reaches MaxRetry. Successive runs are
// ordered by the scheduler, so plain fields need no synchronization.
type retryTask struct {
	*Base
	RetryCount int64 `json:"retry_count"`
	MaxRetry   int64 `json:"max_retry"`
}

func newRetryTask(id string, e time.Time, maxRetry int64) *retryTask {
	return &retryTask{Base: newBase(id, e), MaxRetry: maxRetry}
}

func (t *retryTask) Timeout() time.Duration { return time.Millisecond }

func (t *retryTask) RetryTime() time.Time {
	return time.Now().UTC().Add(100 * time.Millisecond)
}

func (t *retryTask) Execute() (any, bool, error) {
	if t.RetryCount >= t.MaxRetry {
		o.markLast()
		return fmt.Sprintf("retry task %s finished after %d retries",
			t.id, t.RetryCount), false, nil
	}
	o.push(t.id)
	o.markFirst()
	t.RetryCount++
	return fmt.Sprintf("retry task %s, retry %d", t.id, t.RetryCount), true, nil
}

// failTask reports an error on its first run and succeeds afterwards.
type failTask struct {
	*Base
	fails int64
}

func newFailTask(id string, e time.Time) *failTask { return &failTask{Base: newBase(id, e)} }

func (t *failTask) RetryTime() time.Time { return time.Now().UTC().Add(50 * time.Millisecond) }

func (t *failTask) Execute() (any, bool, error) {
	o.push(t.id)
	t.fails++
	if t.fails == 1 {
		return nil, false, fmt.Errorf("task %s failed", t.id)
	}
	return "recovered", false, nil
}

// TestMain silences the go-redis handshake notice that miniredis triggers,
// so that a test failure is the only thing in the output.
func TestMain(m *testing.M) {
	redis.SetLogger(discardLogger{})
	before := runtime.NumGoroutine()
	code := m.Run()
	if code == 0 {
		if n := leakedGoroutines(before); n > 0 {
			fmt.Fprintf(os.Stderr, "sched leaked %d goroutines:\n%s", n, stacks())
			code = 1
		}
	}
	os.Exit(code)
}

// leakedGoroutines reports how many goroutines the tests added and never
// released. Every scheduler loop and task goroutine must be gone once Stop
// returned.
func leakedGoroutines(before int) int {
	for range 100 {
		if runtime.NumGoroutine() <= before {
			return 0
		}
		time.Sleep(20 * time.Millisecond)
	}
	return runtime.NumGoroutine() - before
}

func stacks() []byte {
	buf := make([]byte, 1<<20)
	return buf[:runtime.Stack(buf, true)]
}

type discardLogger struct{}

func (discardLogger) Printf(context.Context, string, ...any) {}
