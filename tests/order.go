// Copyright 2018 Changkun Ou. All rights reserved.
// Use of this source code is governed by a MIT
// license that can be found in the LICENSE file.

package tests

import (
	"sync"
	"time"
)

// O order
var O = Order{}

// Order is used for recording execution order
type Order struct {
	mu    sync.Mutex
	order []string
	first time.Time
	last  time.Time
}

// Push an execution id
func (o *Order) Push(s string) {
	o.mu.Lock()
	o.order = append(o.order, s)
	o.mu.Unlock()
}

// IsFirstZero check if first is zero time
func (o *Order) IsFirstZero() bool {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.first.IsZero()
}

// SetFirst time
func (o *Order) SetFirst(t time.Time) {
	o.mu.Lock()
	o.first = t
	o.mu.Unlock()
}

// GetFirst time
func (o *Order) GetFirst() time.Time {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.first
}

// SetLast time
func (o *Order) SetLast(t time.Time) {
	o.mu.Lock()
	o.last = t
	o.mu.Unlock()
}

// GetLast time
func (o *Order) GetLast() time.Time {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.last
}

// Get order
func (o *Order) Get() []string {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.order
}

// Clear the order
func (o *Order) Clear() {
	o.mu.Lock()
	o.order = []string{}
	o.first = time.Time{}
	o.last = time.Time{}
	o.mu.Unlock()
}
