// Copyright 2018 Changkun Ou. All rights reserved.
// Use of this source code is governed by a MIT
// license that can be found in the LICENSE file.

package sched

import (
	"context"
	"errors"
	"slices"
	"sync/atomic"
	"testing"
	"time"
)

var errInjected = errors.New("sched: injected store failure")

// faultyStore fails the operations that a test switches on, so that the
// error paths of the scheduler are reachable without a broken Redis.
type faultyStore struct {
	store
	failGet   atomic.Bool
	failSet   atomic.Bool
	failDel   atomic.Bool
	failSetNX atomic.Bool
	failKeys  atomic.Bool
	lockTaken atomic.Bool
}

func (s *faultyStore) Get(ctx context.Context, key string) (string, error) {
	if s.failGet.Load() {
		return "", errInjected
	}
	return s.store.Get(ctx, key)
}

func (s *faultyStore) Set(ctx context.Context, key, value string) error {
	if s.failSet.Load() {
		return errInjected
	}
	return s.store.Set(ctx, key, value)
}

func (s *faultyStore) Del(ctx context.Context, key string) error {
	if s.failDel.Load() {
		return errInjected
	}
	return s.store.Del(ctx, key)
}

func (s *faultyStore) SetNX(ctx context.Context, key, value string, ttl time.Duration) (bool, error) {
	if s.failSetNX.Load() {
		return false, errInjected
	}
	if s.lockTaken.Load() {
		return false, nil
	}
	return s.store.SetNX(ctx, key, value, ttl)
}

func (s *faultyStore) Keys(ctx context.Context, prefix string) ([]string, error) {
	if s.failKeys.Load() {
		return nil, errInjected
	}
	return s.store.Keys(ctx, prefix)
}

// newFaultyStore starts an in-process Redis behind a faultyStore.
func newFaultyStore(t *testing.T) *faultyStore {
	t.Helper()
	_, url := newServer(t)
	st, err := newRedisStore(url)
	if err != nil {
		t.Fatalf("newRedisStore: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	return &faultyStore{store: st}
}

func TestRedisStoreRoundTrip(t *testing.T) {
	_, url := newServer(t)
	st, err := newRedisStore(url)
	if err != nil {
		t.Fatalf("newRedisStore: %v", err)
	}
	ctx := t.Context()

	if err := st.Set(ctx, prefixTask+"a", "value"); err != nil {
		t.Fatalf("Set: %v", err)
	}
	if got, err := st.Get(ctx, prefixTask+"a"); err != nil || got != "value" {
		t.Fatalf("Get = %q, %v, want \"value\", nil", got, err)
	}

	ok, err := st.SetNX(ctx, prefixLock+"a", "locked", time.Minute)
	if err != nil || !ok {
		t.Fatalf("SetNX = %v, %v, want true, nil", ok, err)
	}
	if ok, err := st.SetNX(ctx, prefixLock+"a", "locked", time.Minute); err != nil || ok {
		t.Fatalf("second SetNX = %v, %v, want false, nil", ok, err)
	}

	keys, err := st.Keys(ctx, prefixTask)
	if err != nil || !slices.Equal(keys, []string{prefixTask + "a"}) {
		t.Fatalf("Keys = %v, %v", keys, err)
	}
	ids, err := taskIDs(ctx, st)
	if err != nil || !slices.Equal(ids, []string{"a"}) {
		t.Fatalf("taskIDs = %v, %v, want [a]", ids, err)
	}

	if err := st.Del(ctx, prefixTask+"a"); err != nil {
		t.Fatalf("Del: %v", err)
	}
	if _, err := st.Get(ctx, prefixTask+"a"); err == nil {
		t.Fatal("Get after Del must fail")
	}
	if err := st.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
}

func TestSaveAndReadRecord(t *testing.T) {
	st := newFaultyStore(t)
	ctx := t.Context()

	when := time.Now().UTC().Truncate(time.Millisecond)
	if err := saveTask(ctx, st, newTask("r", when)); err != nil {
		t.Fatalf("saveTask: %v", err)
	}
	r, err := readRecord(ctx, st, "r")
	if err != nil {
		t.Fatalf("readRecord: %v", err)
	}
	if !r.Execution.Equal(when) || r.ID != "r" {
		t.Fatalf("record = %+v, want id r at %v", r, when)
	}

	// A task json cannot encode never reaches the store.
	if err := saveTask(ctx, st, &badTask{Base: newBase("bad", when)}); err == nil {
		t.Fatal("saveTask must reject an unencodable task")
	}

	// A missing key and a broken payload both surface as errors.
	if _, err := readRecord(ctx, st, "absent"); err == nil {
		t.Fatal("readRecord of a missing task must fail")
	}
	if err := st.Set(ctx, prefixTask+"broken", "{not json"); err != nil {
		t.Fatalf("Set: %v", err)
	}
	if _, err := readRecord(ctx, st, "broken"); err == nil {
		t.Fatal("readRecord of a broken payload must fail")
	}

	st.failKeys.Store(true)
	if _, err := taskIDs(ctx, st); !errors.Is(err, errInjected) {
		t.Fatalf("taskIDs = %v, want the injected failure", err)
	}
}

func TestRestoreFailure(t *testing.T) {
	st := newFaultyStore(t)
	st.failKeys.Store(true)
	if _, err := start(st, &task{}); !errors.Is(err, errInjected) {
		t.Fatalf("start = %v, want the injected failure", err)
	}
	Stop()
}

func TestLoadRejectsUnusableRecords(t *testing.T) {
	st := newFaultyStore(t)
	s := newSched(st)
	ctx := t.Context()

	if f := s.load("x", nil); f != nil {
		t.Fatal("a nil prototype must not load")
	}
	if f := s.load("x", notPointer{}); f != nil {
		t.Fatal("a non-pointer prototype must not load")
	}
	if f := s.load("absent", &task{}); f != nil {
		t.Fatal("a missing record must not load")
	}

	// A record whose payload has no field of the prototype leaves every
	// exported field zero, which sched refuses to restore.
	if err := saveTask(ctx, st, newNilTask("other", time.Now().UTC())); err != nil {
		t.Fatalf("saveTask: %v", err)
	}
	if err := st.Set(ctx, prefixTask+"other",
		`{"id":"other","execution":"2020-01-01T00:00:00Z","data":{}}`); err != nil {
		t.Fatalf("Set: %v", err)
	}
	if f := s.load("other", &task{}); f != nil {
		t.Fatal("an empty payload must not load")
	}

	// A payload of the wrong shape cannot be decoded at all.
	if err := st.Set(ctx, prefixTask+"scalar",
		`{"id":"scalar","execution":"2020-01-01T00:00:00Z","data":"a string"}`); err != nil {
		t.Fatalf("Set: %v", err)
	}
	if f := s.load("scalar", &task{}); f != nil {
		t.Fatal("a payload of the wrong shape must not load")
	}
}

// notPointer is a Task whose prototype is not a pointer, which sched cannot
// allocate a copy of.
type notPointer struct{}

func (notPointer) ID() string                  { return "" }
func (notPointer) SetID(string)                {}
func (notPointer) Execution() time.Time        { return time.Time{} }
func (notPointer) SetExecution(time.Time)      {}
func (notPointer) Timeout() time.Duration      { return 0 }
func (notPointer) RetryTime() time.Time        { return time.Time{} }
func (notPointer) Execute() (any, bool, error) { return nil, false, nil }

func TestExecuteSkipsUnverifiableTask(t *testing.T) {
	o.clear()
	st := newFaultyStore(t)
	if _, err := start(st); err != nil {
		t.Fatalf("start: %v", err)
	}
	t.Cleanup(Stop)

	f, err := Submit(newTask("unverifiable", time.Now().UTC().Add(150*time.Millisecond)))
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}
	// The record disappears before the task runs, so the scheduler cannot
	// confirm the execution time and drops the task.
	st.failGet.Store(true)
	got, ok := f.Get().(error)
	if !ok || !errors.Is(got, ErrTaskUnverifiable) || !errors.Is(got, errInjected) {
		t.Fatalf("future value = %v, want ErrTaskUnverifiable wrapping the store failure", f.Get())
	}
	wantOrder(t, nil)
}

func TestExecuteSkipsWhenLockFails(t *testing.T) {
	o.clear()
	st := newFaultyStore(t)
	st.failSetNX.Store(true)
	if _, err := start(st); err != nil {
		t.Fatalf("start: %v", err)
	}
	t.Cleanup(Stop)

	f, err := Submit(newTask("unlockable", time.Now().UTC()))
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}
	got, ok := f.Get().(error)
	if !ok || !errors.Is(got, errInjected) {
		t.Fatalf("future value = %v, want the injected lock failure", f.Get())
	}
	wantOrder(t, nil)
}

func TestExecuteToleratesFailingDeletes(t *testing.T) {
	o.clear()
	st := newFaultyStore(t)
	if _, err := start(st); err != nil {
		t.Fatalf("start: %v", err)
	}
	t.Cleanup(Stop)

	st.failDel.Store(true)
	f, err := Submit(newTask("undeletable", time.Now().UTC()))
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}
	if got, ok := f.Get().(string); !ok || got == "" {
		t.Fatalf("future value = %v, want a result despite the failing delete", f.Get())
	}
}

func TestRescheduleSurvivesFailingSave(t *testing.T) {
	o.clear()
	st := newFaultyStore(t)
	if _, err := start(st); err != nil {
		t.Fatalf("start: %v", err)
	}
	t.Cleanup(Stop)

	f, err := Submit(newFailTask("retry-nosave", time.Now().UTC()))
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}
	// The retry cannot be persisted, but it still has to run.
	st.failSet.Store(true)
	if got := f.Get(); got != "recovered" {
		t.Fatalf("future value = %v, want \"recovered\"", got)
	}
}
