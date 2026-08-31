// Copyright 2018 Changkun Ou. All rights reserved.
// Use of this source code is governed by a MIT
// license that can be found in the LICENSE file.

package sched

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
)

// newServer starts an in-process Redis and returns its URL.
func newServer(t *testing.T) (*miniredis.Miniredis, string) {
	t.Helper()
	mr := miniredis.RunT(t)
	return mr, "redis://" + mr.Addr() + "/0"
}

// setup starts a scheduler on a fresh in-process Redis.
func setup(t *testing.T, prototypes ...Task) *miniredis.Miniredis {
	t.Helper()
	o.clear()
	mr, url := newServer(t)
	if _, err := Init(url, prototypes...); err != nil {
		t.Fatalf("Init: %v", err)
	}
	t.Cleanup(Stop)
	return mr
}

func wantOrder(t *testing.T, want []string) {
	t.Helper()
	if got := o.get(); !slices.Equal(got, want) {
		t.Fatalf("execution order = %v, want %v", got, want)
	}
}

func TestInitBadURL(t *testing.T) {
	if _, err := Init("rdis://127.0.0.1:6323/123123"); err == nil {
		t.Fatal("Init with an invalid URL must fail")
	}
}

func TestUninitialized(t *testing.T) {
	Stop() // drop any scheduler a previous test left behind
	sched0.Store(nil)

	if _, err := Submit(newTask("x", time.Now())); !errors.Is(err, ErrNotInitialized) {
		t.Fatalf("Submit before Init = %v, want ErrNotInitialized", err)
	}
	if _, err := Trigger(newTask("x", time.Now())); !errors.Is(err, ErrNotInitialized) {
		t.Fatalf("Trigger before Init = %v, want ErrNotInitialized", err)
	}
	// These must not panic without a scheduler.
	Pause()
	Resume()
	Wait()
	Stop()
}

func TestScheduleOrder(t *testing.T) {
	setup(t)

	start := time.Now().UTC()
	futures := make([]Future, 20)
	want := make([]string, 20)
	for i := range futures {
		id := fmt.Sprintf("task-%d", i)
		want[i] = id
		f, err := Submit(newTask(id, start.Add(time.Duration(i)*20*time.Millisecond)))
		if err != nil {
			t.Fatalf("Submit: %v", err)
		}
		futures[i] = f
	}
	for _, f := range futures {
		if _, ok := f.Get().(string); !ok {
			t.Fatalf("future value = %v, want a string", f.Get())
		}
	}
	wantOrder(t, want)
}

func TestScheduleReverseOrder(t *testing.T) {
	setup(t)

	start := time.Now().UTC().Add(200 * time.Millisecond)
	var futures []Future
	var want []string
	// Submit in reverse, so the queue has to reorder every insertion.
	for i := 9; i >= 0; i-- {
		id := fmt.Sprintf("task-%d", i)
		f, err := Submit(newTask(id, start.Add(time.Duration(i)*30*time.Millisecond)))
		if err != nil {
			t.Fatalf("Submit: %v", err)
		}
		futures = append(futures, f)
	}
	for i := range 10 {
		want = append(want, fmt.Sprintf("task-%d", i))
	}
	for _, f := range futures {
		f.Get()
	}
	wantOrder(t, want)
}

func TestSubmitTwiceShares(t *testing.T) {
	setup(t)

	start := time.Now().UTC()
	first, err := Submit(newTask("dup", start.Add(500*time.Millisecond)))
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}
	// The same identifier at an earlier time moves the queued task and
	// resolves both futures.
	second, err := Submit(newTask("dup", start.Add(50*time.Millisecond)))
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}
	if got, want := first.Get(), second.Get(); got != want {
		t.Fatalf("futures differ: %v vs %v", got, want)
	}
	wantOrder(t, []string{"dup"})
}

func TestTriggerRunsNow(t *testing.T) {
	setup(t)

	f, err := Trigger(newTask("now", time.Now().UTC().Add(time.Hour)))
	if err != nil {
		t.Fatalf("Trigger: %v", err)
	}
	select {
	case <-f.Done():
	case <-time.After(2 * time.Second):
		t.Fatal("Trigger did not run the task immediately")
	}
	wantOrder(t, []string{"now"})
}

func TestConcurrentSubmitAndTrigger(t *testing.T) {
	setup(t)

	start := time.Now().UTC()
	if _, err := Submit(newTask("task-1", start.Add(300*time.Millisecond))); err != nil {
		t.Fatalf("Submit: %v", err)
	}

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		f, err := Submit(newTask("task-2", start.Add(600*time.Millisecond)))
		if err != nil {
			t.Error(err)
			return
		}
		f.Get()
	}()
	go func() {
		defer wg.Done()
		f, err := Trigger(newTask("task-1", start))
		if err != nil {
			t.Error(err)
			return
		}
		f.Get()
	}()
	wg.Wait()
	wantOrder(t, []string{"task-1", "task-2"})
}

func TestNilResult(t *testing.T) {
	setup(t)

	f, err := Submit(newNilTask("nil-task", time.Now().UTC()))
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}
	got, ok := f.Get().(string)
	if !ok || got == "" {
		t.Fatalf("future value = %v, want a placeholder string", f.Get())
	}
}

func TestPanicReachesFuture(t *testing.T) {
	setup(t)

	f, err := Submit(newPanicTask("boom", time.Now().UTC()))
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}
	err, ok := f.Get().(error)
	if !ok {
		t.Fatalf("future value = %v, want an error", f.Get())
	}
	if got := err.Error(); got == "" {
		t.Fatal("panic error must describe the task")
	}
	wantOrder(t, []string{"boom"})
}

func TestRetry(t *testing.T) {
	setup(t)

	f, err := Submit(newRetryTask("retry", time.Now().UTC(), 3))
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}
	f.Get()
	wantOrder(t, []string{"retry", "retry", "retry"})
	if span := o.span(); span < 200*time.Millisecond {
		t.Fatalf("retries took %v, want at least 200ms apart", span)
	}
}

func TestRetryAfterError(t *testing.T) {
	setup(t)

	f, err := Submit(newFailTask("fail", time.Now().UTC()))
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}
	if got := f.Get(); got != "recovered" {
		t.Fatalf("future value = %v, want \"recovered\"", got)
	}
	wantOrder(t, []string{"fail", "fail"})
}

func TestStoreHoldsLaterExecution(t *testing.T) {
	mr := setup(t)

	start := time.Now().UTC()
	f, err := Submit(newTask("late", start.Add(200*time.Millisecond)))
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}
	// Another replica postpones the task in the store. This replica must
	// notice at execution time and reschedule instead of running early.
	postpone(t, mr, "late", 400*time.Millisecond)

	time.Sleep(300 * time.Millisecond)
	wantOrder(t, nil)

	f.Get()
	wantOrder(t, []string{"late"})
}

// postpone rewrites the persisted execution time of a task.
func postpone(t *testing.T, mr *miniredis.Miniredis, id string, d time.Duration) {
	t.Helper()
	raw, err := mr.Get(prefixTask + id)
	if err != nil {
		t.Fatalf("read record: %v", err)
	}
	var r record
	if err := json.Unmarshal([]byte(raw), &r); err != nil {
		t.Fatalf("decode record: %v", err)
	}
	r.Execution = r.Execution.Add(d)
	out, err := json.Marshal(&r)
	if err != nil {
		t.Fatalf("encode record: %v", err)
	}
	if err := mr.Set(prefixTask+id, string(out)); err != nil {
		t.Fatalf("write record: %v", err)
	}
}

func TestPauseAndResume(t *testing.T) {
	setup(t)

	f, err := Submit(newTask("paused", time.Now().UTC().Add(100*time.Millisecond)))
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}
	Pause()
	time.Sleep(400 * time.Millisecond)
	wantOrder(t, nil)

	Resume()
	f.Get()
	wantOrder(t, []string{"paused"})
}

func TestWaitDrainsQueue(t *testing.T) {
	setup(t)

	start := time.Now().UTC()
	for i := range 5 {
		if _, err := Submit(newTask(fmt.Sprintf("w-%d", i),
			start.Add(time.Duration(i)*20*time.Millisecond))); err != nil {
			t.Fatalf("Submit: %v", err)
		}
	}
	// Several waiters must all wake up.
	var wg sync.WaitGroup
	for range 3 {
		wg.Add(1)
		go func() { defer wg.Done(); Wait() }()
	}
	wg.Wait()
	if got := len(o.get()); got != 5 {
		t.Fatalf("Wait returned with %d of 5 tasks done", got)
	}
}

func TestStopWaitsForRunningTask(t *testing.T) {
	o.clear()
	_, url := newServer(t)
	if _, err := Init(url); err != nil {
		t.Fatalf("Init: %v", err)
	}

	f, err := Submit(newTask("stopping", time.Now().UTC().Add(100*time.Millisecond)))
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}
	f.Get()
	Stop()
	Stop() // idempotent
	wantOrder(t, []string{"stopping"})
}

func TestStopLeavesQueuedTaskForRecovery(t *testing.T) {
	o.clear()
	mr, url := newServer(t)
	if _, err := Init(url); err != nil {
		t.Fatalf("Init: %v", err)
	}
	if _, err := Submit(newTask("survivor", time.Now().UTC().Add(time.Hour))); err != nil {
		t.Fatalf("Submit: %v", err)
	}
	Stop()

	// The record outlives the scheduler and comes back on the next Init.
	if _, err := mr.Get(prefixTask + "survivor"); err != nil {
		t.Fatalf("queued task must stay in the store: %v", err)
	}
	futures, err := Init(url, &task{})
	if err != nil {
		t.Fatalf("Init: %v", err)
	}
	t.Cleanup(Stop)
	if len(futures) != 1 {
		t.Fatalf("recovered %d futures, want 1", len(futures))
	}
}

func TestRecover(t *testing.T) {
	o.clear()
	mr, url := newServer(t)

	start := time.Now().UTC()
	want := make([]string, 5)
	st, err := newRedisStore(url)
	if err != nil {
		t.Fatalf("newRedisStore: %v", err)
	}
	for i := range want {
		id := fmt.Sprintf("task-%d", i)
		want[i] = id
		if err := saveTask(t.Context(), st,
			newTask(id, start.Add(time.Duration(i)*20*time.Millisecond))); err != nil {
			t.Fatalf("saveTask: %v", err)
		}
	}
	if err := st.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	_ = mr

	futures, err2 := Init(url, &task{})
	if err2 != nil {
		t.Fatalf("Init: %v", err2)
	}
	t.Cleanup(Stop)
	if len(futures) != len(want) {
		t.Fatalf("recovered %d futures, want %d", len(futures), len(want))
	}
	for _, f := range futures {
		f.Get()
	}
	wantOrder(t, want)
}

func TestInitReplacesRunningScheduler(t *testing.T) {
	o.clear()
	_, url := newServer(t)
	if _, err := Init(url); err != nil {
		t.Fatalf("Init: %v", err)
	}
	first := current()
	if _, err := Init(url); err != nil {
		t.Fatalf("Init again: %v", err)
	}
	t.Cleanup(Stop)
	if current() == first {
		t.Fatal("Init must install a new scheduler")
	}
	select {
	case <-first.done:
	case <-time.After(2 * time.Second):
		t.Fatal("Init must stop the scheduler it replaces")
	}
}

func TestLockHeldByAnotherReplica(t *testing.T) {
	mr := setup(t)

	id := "locked"
	if err := mr.Set(prefixLock+id, "locked"); err != nil {
		t.Fatalf("take lock: %v", err)
	}
	f, err := Submit(newTask(id, time.Now().UTC()))
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}
	got, ok := f.Get().(error)
	if !ok || !errors.Is(got, ErrTaskClaimed) {
		t.Fatalf("future value = %v, want ErrTaskClaimed", f.Get())
	}
	wantOrder(t, nil)
}

func TestSubmitStoreFailure(t *testing.T) {
	o.clear()
	_, url := newServer(t)
	st, err := newRedisStore(url)
	if err != nil {
		t.Fatalf("newRedisStore: %v", err)
	}
	fs := &faultyStore{store: st}
	fs.failSet.Store(true)
	if _, err := start(fs); err != nil {
		t.Fatalf("start: %v", err)
	}
	t.Cleanup(Stop)

	if _, err := Submit(newTask("nope", time.Now().UTC())); err == nil {
		t.Fatal("Submit must report a store failure")
	}
}

func TestUnmarshalableTask(t *testing.T) {
	setup(t)

	if _, err := Submit(&badTask{Base: newBase("bad", time.Now().UTC())}); err == nil {
		t.Fatal("Submit must reject a task that json cannot encode")
	}
}

// badTask cannot be encoded, so it cannot be persisted or scheduled.
type badTask struct {
	*Base
	Ch chan int `json:"ch"`
}

func (t *badTask) Execute() (any, bool, error) { return nil, false, nil }

func TestConcurrentSubmitters(t *testing.T) {
	setup(t)

	const n = 64
	start := time.Now().UTC()
	futures := make([]Future, n)
	var wg sync.WaitGroup
	for i := range n {
		wg.Add(1)
		go func() {
			defer wg.Done()
			f, err := Submit(newTask(fmt.Sprintf("c-%d", i), start))
			if err != nil {
				t.Error(err)
				return
			}
			futures[i] = f
		}()
	}
	wg.Wait()
	for i, f := range futures {
		if f == nil {
			t.Fatalf("submission %d lost", i)
		}
		f.Get()
	}
	if got := len(o.get()); got != n {
		t.Fatalf("%d of %d tasks ran", got, n)
	}
}

func TestFutureResolvesOnce(t *testing.T) {
	f := newFuture()
	f.put("first")
	f.put("second")
	if got := f.Get(); got != "first" {
		t.Fatalf("Get() = %v, want \"first\"", got)
	}
	select {
	case <-f.Done():
	default:
		t.Fatal("Done must be closed after put")
	}
}

func TestContextCancelsFutureWait(t *testing.T) {
	f := newFuture()
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	select {
	case <-f.Done():
		t.Fatal("future must stay pending")
	case <-ctx.Done():
	}
}

func TestStopReleasesQueuedFutures(t *testing.T) {
	o.clear()
	_, url := newServer(t)
	if _, err := Init(url); err != nil {
		t.Fatalf("Init: %v", err)
	}
	f, err := Submit(newTask("never", time.Now().UTC().Add(time.Hour)))
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}
	s := current()
	Stop()

	got, ok := f.Get().(error)
	if !ok || !errors.Is(got, ErrStopped) {
		t.Fatalf("future value = %v, want ErrStopped", f.Get())
	}
	if _, err := s.submit(newTask("after", time.Now().UTC())); !errors.Is(err, ErrNotInitialized) {
		t.Fatalf("submit after Stop = %v, want ErrNotInitialized", err)
	}
}

func TestAbandonReleasesQueuedAndPendingFutures(t *testing.T) {
	st := newFaultyStore(t)
	s := newSched(st)

	queued := newEntry("queued", time.Now().UTC())
	s.queued.Add(1)
	s.tasks.push(queued)

	pending := newEntry("pending", time.Now().UTC())
	s.pending.Add(1)
	s.intake.push(pending)

	s.abandon()

	for _, e := range []*entry{queued, pending} {
		got, ok := e.futures[0].Get().(error)
		if !ok || !errors.Is(got, ErrStopped) {
			t.Fatalf("%s future = %v, want ErrStopped", e.task.ID(), e.futures[0].Get())
		}
	}
	if s.outstanding() != 0 {
		t.Fatalf("outstanding = %d, want 0", s.outstanding())
	}
}

func TestConcurrentStop(t *testing.T) {
	o.clear()
	_, url := newServer(t)
	if _, err := Init(url); err != nil {
		t.Fatalf("Init: %v", err)
	}
	s := current()

	var wg sync.WaitGroup
	for range 4 {
		wg.Add(1)
		go func() { defer wg.Done(); s.shutdown() }()
	}
	wg.Wait()
	Stop()

	select {
	case <-s.done:
	default:
		t.Fatal("the scheduler loop must have returned")
	}
}

func TestAwaitReturnsAfterTheLoopStops(t *testing.T) {
	o.clear()
	_, url := newServer(t)
	if _, err := Init(url); err != nil {
		t.Fatalf("Init: %v", err)
	}
	s := current()
	Stop()

	// The condition never holds, so await can only return on s.done.
	done := make(chan struct{})
	go func() { defer close(done); s.await(func() bool { return false }) }()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("await must return once the scheduler stopped")
	}
}
