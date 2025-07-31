// Copyright 2021, 2026 The Alpaca Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import "sync"

type proxyRotation struct {
	order []string        // hosts ordered by demotion time (most recently demoted at end)
	index map[string]int  // host -> position in order
	mux   sync.Mutex
}

func newProxyRotation() *proxyRotation {
	return &proxyRotation{
		order: []string{},
		index: map[string]int{},
	}
}

func (r *proxyRotation) demote(host string) {
	r.mux.Lock()
	defer r.mux.Unlock()
	if _, ok := r.index[host]; ok {
		// Remove from current position
		r.removeFromOrder(host)
	}
	// Append to end (most recently demoted)
	r.order = append(r.order, host)
	r.index[host] = len(r.order) - 1
}

// less returns true if a should be preferred over b.
// A host not in the rotation (never demoted) is preferred over one that has been demoted.
// Among demoted hosts, the one demoted less recently (lower index) is preferred.
func (r *proxyRotation) less(a, b string) bool {
	r.mux.Lock()
	defer r.mux.Unlock()
	ai, aOk := r.index[a]
	bi, bOk := r.index[b]
	if !aOk && !bOk {
		return false // preserve original order
	}
	if !aOk {
		return true // a never demoted, prefer a
	}
	if !bOk {
		return false // b never demoted, prefer b
	}
	return ai < bi
}

func (r *proxyRotation) removeFromOrder(host string) {
	pos := r.index[host]
	r.order = append(r.order[:pos], r.order[pos+1:]...)
	delete(r.index, host)
	// Reindex everything after the removed position
	for i := pos; i < len(r.order); i++ {
		r.index[r.order[i]] = i
	}
}
