// Copyright 2018 Changkun Ou. All rights reserved.
// Use of this source code is governed by a MIT
// license that can be found in the LICENSE file.

package tests

import "sync"

// O order
var O = Order{}

// Order is used for recording execution order
type Order struct {
	mu    sync.Mutex
	order []string
}

// Push an execution id
func (o *Order) Push(s string) {
	o.mu.Lock()
	o.order = append(o.order, s)
	o.mu.Unlock()
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
	o.mu.Unlock()
}
