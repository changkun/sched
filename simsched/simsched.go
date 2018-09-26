// Copyright 2018 Changkun Ou. All rights reserved.
// Use of this source code is governed by a MIT
// license that can be found in the LICENSE file.

package simsched

import (
	"sync"
	"time"

	"github.com/changkun/sched/internal/pq"
	"github.com/changkun/sched/task"
)

var (
	worker        *Scheduler
	schedulerOnce sync.Once
)

// Scheduler is the actual scheduler for task scheduling
type Scheduler struct {
	mu    *sync.Mutex
	timer *time.Timer
	queue *pq.TaskQueue
}

// New creates a temporary task scheduler
func New() *Scheduler {
	schedulerOnce.Do(func() {
		worker = &Scheduler{
			mu:    &sync.Mutex{},
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
		New().simpleschd(ts[i])
	}
}

// Boot given tasks immediately without persistance guarantee
func Boot(ts ...task.Interface) {
	New().Boot(ts...)
}

// Boot given tasks immediately without persistance guarantee
func (s *Scheduler) Boot(ts ...task.Interface) {
	for i := range ts {
		s.simplelaunch(ts[i])
	}
}

func (s *Scheduler) simplelaunch(t task.Interface) {
	t.SetExecution(time.Now().UTC())
	s.Submit(t)
}

func (s *Scheduler) setTimer(duration time.Duration) {
	s.mu.Lock()
	s.timer = time.NewTimer(duration)
	s.mu.Unlock()
}

func (s *Scheduler) getTimer() *time.Timer {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.timer
}

func (s *Scheduler) simpleschd(t task.Interface) {
	s.simplepause()
	defer s.simpleresume()

	// if priority is able to be update
	if s.queue.Update(t) {
		return
	}

	s.queue.Push(t)
}
func (s *Scheduler) simplepause() {
	s.mu.Lock()
	if s.timer != nil {
		s.timer.Stop()
	}
	s.mu.Unlock()
}
func (s *Scheduler) simpleresume() {
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
		s.simpleresume()
		s.simplearrival(t)
	}()
}

func (s *Scheduler) simplearrival(t task.Interface) {
	s.simpleexecute(t)
}

func (s *Scheduler) simpleretry(t task.Interface) {
	t.SetExecution(t.GetExecution().Add(t.GetRetryDuration()))
	s.Submit(t)
}

func (s *Scheduler) simpleexecute(t task.Interface) {
	if t.GetExecution().Before(time.Now().UTC()) {
		retry, err := t.Execute()
		if retry || err != nil {
			s.simpleretry(t)
			return
		}
		return
	}
	s.Submit(t)
}
