// Copyright 2018 Changkun Ou. All rights reserved.
// Use of this source code is governed by a MIT
// license that can be found in the LICENSE file.

package simsched

import (
	"sync"
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/changkun/sched/internal/pq"
	"github.com/changkun/sched/task"
)

var (
	worker        *Scheduler
	schedulerOnce sync.Once
)

// Scheduler is the actual scheduler for task scheduling
type Scheduler struct {
	timer unsafe.Pointer
	queue *pq.TaskQueue
}

// New creates a temporary task scheduler
func New() *Scheduler {
	schedulerOnce.Do(func() {
		worker = &Scheduler{
			timer: nil,
			queue: pq.NewTaskQueue(),
		}
	})
	return worker
}

// Submit task without persistance guarantee
func Submit(ts ...task.Interface) {
	New().Submit(ts...)
}

// Submit task without persistance guarantee
func (s *Scheduler) Submit(ts ...task.Interface) {
	for i := range ts {
		New().simschd(ts[i])
	}
}

// Boot given tasks immediately without persistance guarantee
func Boot(ts ...task.Interface) {
	New().Boot(ts...)
}

// Boot given tasks immediately without persistance guarantee
func (s *Scheduler) Boot(ts ...task.Interface) {
	for i := range ts {
		s.simlaunch(ts[i])
	}
}

func (s *Scheduler) simlaunch(t task.Interface) {
	t.SetExecution(time.Now().UTC())
	s.Submit(t)
}

func (s *Scheduler) setTimer(duration time.Duration) {
	atomic.StorePointer(&s.timer, unsafe.Pointer(time.NewTimer(duration)))
}

func (s *Scheduler) getTimer() *time.Timer {
	return (*time.Timer)(atomic.LoadPointer(&s.timer))
}

func (s *Scheduler) simschd(t task.Interface) {
	s.simpause()
	defer s.simresume()

	// if priority is able to be update
	if s.queue.Update(t) {
		return
	}

	s.queue.Push(t)
}
func (s *Scheduler) simpause() {
	// fast path.
	if atomic.LoadPointer(&s.timer) == nil {
		return
	}

	(*time.Timer)(atomic.LoadPointer(&s.timer)).Stop()
}
func (s *Scheduler) simresume() {
	t := s.queue.Peek()
	if t == nil {
		return
	}
	s.setTimer(t.GetExecution().Sub(time.Now().UTC()))
	go func() {
		<-s.getTimer().C
		t := s.queue.Pop()
		if t == nil {
			return
		}
		s.simresume()
		s.simarrival(t)
	}()
}

func (s *Scheduler) simarrival(t task.Interface) {
	s.simexecute(t)
}

func (s *Scheduler) simretry(t task.Interface) {
	t.SetExecution(t.GetExecution().Add(t.GetRetryDuration()))
	s.Submit(t)
}

func (s *Scheduler) simexecute(t task.Interface) {
	if t.GetExecution().Before(time.Now().UTC()) {
		retry, err := t.Execute()
		if retry || err != nil {
			s.simretry(t)
			return
		}
		return
	}
	s.Submit(t)
}
