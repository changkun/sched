// Copyright 2018 Changkun Ou. All rights reserved.
// Use of this source code is governed by a MIT
// license that can be found in the LICENSE file.

package sched

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"sync/atomic"
	"time"
)

// ErrNotInitialized is returned by Submit and Trigger before Init succeeded.
var ErrNotInitialized = errors.New("sched: not initialized, call Init first")

// sched0 is the scheduler the package-level API drives. Init installs it and
// Stop removes it.
var sched0 atomic.Pointer[sched]

func current() *sched { return sched0.Load() }

// Init connects sched to a Redis instance and restores the tasks that a
// previous run left behind.
//
// url is a Redis connection string, for example
// "redis://127.0.0.1:6379/0". Every argument in tasks is a prototype: an
// empty value of a task type that sched must be able to recover. Init
// returns one Future per recovered task.
//
// Calling Init again replaces the running scheduler.
func Init(url string, tasks ...Task) ([]Future, error) {
	s, err := newRedisStore(url)
	if err != nil {
		return nil, err
	}
	return start(s, tasks...)
}

// start installs a scheduler on the given store. Init wraps it; tests use it
// to inject a store.
func start(st store, tasks ...Task) ([]Future, error) {
	s := newSched(st)
	if old := sched0.Swap(s); old != nil {
		old.shutdown()
	}
	go s.run()
	return s.restore(tasks...)
}

// Submit schedules t to run at t.Execution() and returns the Future of its
// result. Submitting an identifier that is already scheduled does not
// duplicate the task; it moves the existing one to the new execution time
// and returns a Future that resolves with it.
func Submit(t Task) (Future, error) {
	s := current()
	if s == nil {
		return nil, ErrNotInitialized
	}
	return s.submit(t)
}

// Trigger schedules t to run now and returns the Future of its result.
func Trigger(t Task) (Future, error) {
	s := current()
	if s == nil {
		return nil, ErrNotInitialized
	}
	t.SetExecution(time.Now().UTC())
	return s.submit(t)
}

// Pause stops sched from starting new tasks. Tasks that already run keep
// running. Pause and Resume nest: Resume starts the scheduler again after as
// many calls as Pause received.
func Pause() {
	if s := current(); s != nil {
		s.pausing.Add(1)
		s.wakeup()
	}
}

// Resume undoes one Pause.
func Resume() {
	if s := current(); s != nil {
		s.pausing.Add(-1)
		s.wakeup()
	}
}

// Wait blocks until no task is queued and no task runs. It never returns
// while the scheduler is paused with work outstanding.
func Wait() {
	if s := current(); s != nil {
		s.await(func() bool { return s.outstanding() == 0 })
	}
}

// Stop shuts the scheduler down after the tasks that already started have
// finished, and closes the connection to the store. Queued tasks stay in the
// store and come back on the next Init. Stop is safe to call more than once.
func Stop() {
	if s := current(); s != nil {
		s.shutdown()
		sched0.CompareAndSwap(s, nil)
	}
}

// sched schedules tasks with a single timer and a single priority queue.
//
// One goroutine, run, owns the queue and the timer. Everything else reaches
// it through intake, a wait-free multi-producer queue, and wakeup, a
// non-blocking signal. No goroutine ever waits for a lock to schedule a
// task: Submit, Trigger, Pause and Resume all finish in a bounded number of
// atomic steps.
type sched struct {
	// pausing counts unmatched Pause calls.
	pausing atomic.Int64
	// pending counts entries handed to intake that run has not absorbed.
	// queued counts entries in the priority queue, running counts tasks
	// that execute. pending + queued + running is the work outstanding;
	// the transfers between them always raise the destination before they
	// lower the source, so an observer never sees a spurious zero.
	pending atomic.Int64
	queued  atomic.Int64
	running atomic.Int64
	stopped atomic.Bool

	intake   *intake
	tasks    *queue // owned by run
	store    store
	progress *broadcast

	wake chan struct{} // capacity 1, signals run
	quit chan struct{} // closed to end run
	done chan struct{} // closed when run returned
}

func newSched(st store) *sched {
	return &sched{
		intake:   newIntake(),
		tasks:    newQueue(),
		store:    st,
		progress: newBroadcast(),
		wake:     make(chan struct{}, 1),
		quit:     make(chan struct{}),
		done:     make(chan struct{}),
	}
}

// wakeup asks run to look at the queue again. It never blocks: a signal that
// is already waiting is as good as a second one.
func (s *sched) wakeup() {
	select {
	case s.wake <- struct{}{}:
	default:
	}
}

func (s *sched) paused() bool { return s.pausing.Load() > 0 }

func (s *sched) outstanding() int64 {
	return s.pending.Load() + s.queued.Load() + s.running.Load()
}

// await blocks until cond holds or the scheduler stops.
func (s *sched) await(cond func() bool) {
	for !cond() {
		// Take the channel before the second check, so a signal that
		// arrives in between closes the channel we then wait on.
		changed := s.progress.wait()
		if cond() {
			return
		}
		select {
		case <-changed:
		case <-s.done:
			return
		}
	}
}

// run is the scheduler loop. It is the only goroutine that touches s.tasks.
func (s *sched) run() {
	defer close(s.done)

	// Go 1.23 and later give timers an unbuffered channel: after Stop or
	// Reset no stale value can arrive, so the loop never drains timer.C.
	timer := time.NewTimer(time.Hour)
	timer.Stop()
	defer timer.Stop()

	for {
		for {
			e, ok := s.intake.pop()
			if !ok {
				break
			}
			s.absorb(e)
		}

		var fire <-chan time.Time
		if !s.paused() {
			now := time.Now().UTC()
			for {
				head := s.tasks.peek()
				if head == nil || head.when.After(now) {
					break
				}
				s.dispatch(s.tasks.pop())
			}
			if head := s.tasks.peek(); head != nil {
				timer.Reset(head.when.Sub(now))
				fire = timer.C
			}
		}

		select {
		case <-fire:
		case <-s.wake:
		case <-s.quit:
			return
		}
		if fire != nil {
			timer.Stop()
		}
	}
}

// absorb moves one submitted entry into the queue. A task that is already
// queued keeps its place in the queue and collects the new Future.
func (s *sched) absorb(e *entry) {
	if old := s.tasks.find(e.task.ID()); old != nil {
		old.futures = append(old.futures, e.futures...)
		old.task = e.task
		s.tasks.update(old, e.when)
		s.pending.Add(-1)
		s.progress.fire()
		return
	}
	s.queued.Add(1)
	s.tasks.push(e)
	s.pending.Add(-1)
	s.progress.fire()
}

// dispatch hands a due entry to its own goroutine, so that a slow task never
// delays the scheduler loop.
func (s *sched) dispatch(e *entry) {
	s.running.Add(1)
	s.queued.Add(-1)
	go s.arrive(e)
}

// arrive takes the distributed lock of the task and runs it. If another
// replica holds the lock, this replica drops the task.
func (s *sched) arrive(e *entry) {
	defer func() {
		s.running.Add(-1)
		s.progress.fire()
	}()

	ctx := context.Background()
	ok, err := s.store.SetNX(ctx, prefixLock+e.task.ID(), "locked", e.task.Timeout())
	if err != nil || !ok {
		return
	}
	s.execute(ctx, e)
	// The lock also expires on its own, so a failure to release it delays
	// the next run at worst.
	_ = s.store.Del(ctx, prefixLock+e.task.ID())
}

// execute runs the task and publishes its result.
func (s *sched) execute(ctx context.Context, e *entry) {
	defer func() {
		if r := recover(); r != nil {
			e.put(fmt.Errorf("sched: task %s panicked: %v", e.task.ID(), r))
		}
	}()

	// Another replica may have moved the task since this one queued it.
	// The store holds the truth, and the later of the two times wins.
	r, err := readRecord(ctx, s.store, e.task.ID())
	if err != nil {
		return
	}
	execution := e.task.Execution()
	if execution.Before(r.Execution) {
		execution = r.Execution
	}
	if execution.After(time.Now().UTC()) {
		s.reschedule(ctx, e, execution)
		return
	}

	result, retry, err := e.task.Execute()
	if retry || err != nil {
		s.reschedule(ctx, e, e.task.RetryTime())
		return
	}
	if result == nil {
		result = fmt.Sprintf("sched: task %s returned no result", e.task.ID())
	}
	e.put(result)
	// A failure here leaves a task that the next Init recovers and runs a
	// second time. The task itself has to tolerate that.
	_ = s.store.Del(ctx, prefixTask+e.task.ID())
}

// reschedule puts a task back into the queue at a new time. It raises
// pending before arrive lowers running, so the work stays visible to Wait.
func (s *sched) reschedule(ctx context.Context, e *entry, when time.Time) {
	e.task.SetExecution(when)
	e.when = when
	// Schedule even if the store rejects the write: dropping the task
	// would be worse than running it without a durable record.
	_ = saveTask(ctx, s.store, e.task)
	s.pending.Add(1)
	s.intake.push(e)
	s.wakeup()
}

// submit persists the task and hands it to the scheduler loop.
func (s *sched) submit(t Task) (Future, error) {
	if err := saveTask(context.Background(), s.store, t); err != nil {
		return nil, err
	}
	f := newFuture()
	s.pending.Add(1)
	s.intake.push(&entry{task: t, when: t.Execution(), futures: []*future{f}})
	s.wakeup()
	return f, nil
}

// shutdown pauses the scheduler, waits for the running tasks, ends the loop
// and closes the store.
func (s *sched) shutdown() {
	if !s.stopped.CompareAndSwap(false, true) {
		<-s.done
		return
	}
	s.pausing.Add(1)
	s.wakeup()
	s.await(func() bool { return s.running.Load() == 0 })
	s.pausing.Add(-1)
	close(s.quit)
	<-s.done
	_ = s.store.Close()
}

// restore reads the persisted tasks and queues them again. Every argument is
// a prototype: an empty value of a task type, used to rebuild the concrete
// type that json cannot infer.
func (s *sched) restore(prototypes ...Task) ([]Future, error) {
	ids, err := taskIDs(context.Background(), s.store)
	if err != nil {
		return nil, err
	}
	var futures []Future
	for _, p := range prototypes {
		for _, id := range ids {
			if f := s.load(id, p); f != nil {
				futures = append(futures, f)
			}
		}
	}
	return futures, nil
}

// load rebuilds one persisted task with the type of the prototype. It
// returns nil if the record does not belong to that type.
func (s *sched) load(id string, prototype Task) Future {
	t := reflect.TypeOf(prototype)
	if t == nil || t.Kind() != reflect.Pointer {
		return nil
	}
	r, err := readRecord(context.Background(), s.store, id)
	if err != nil {
		return nil
	}
	data, err := json.Marshal(r.Data)
	if err != nil {
		return nil
	}
	v, ok := reflect.New(t.Elem()).Interface().(Task)
	if !ok {
		return nil
	}
	if err := json.Unmarshal(data, v); err != nil {
		return nil
	}
	// A record of a different task type leaves every exported field at
	// its zero value; such a task cannot be restored and is skipped.
	if reflect.ValueOf(v).Elem().IsZero() {
		return nil
	}
	v.SetID(id)
	v.SetExecution(r.Execution)

	f := newFuture()
	s.pending.Add(1)
	s.intake.push(&entry{task: v, when: r.Execution, futures: []*future{f}})
	s.wakeup()
	return f
}
