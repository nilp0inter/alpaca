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

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDemotedProxyMovesToEnd(t *testing.T) {
	r := newProxyRotation()
	r.demote("foo")
	// foo is demoted, bar is not — bar should be preferred
	assert.True(t, r.less("bar", "foo"))
	assert.False(t, r.less("foo", "bar"))
}

func TestDemoteOrder(t *testing.T) {
	r := newProxyRotation()
	r.demote("foo")
	r.demote("bar")
	// foo was demoted first (lower index), so foo is preferred over bar
	assert.True(t, r.less("foo", "bar"))
	assert.False(t, r.less("bar", "foo"))
}

func TestRedemoteMovesToEnd(t *testing.T) {
	r := newProxyRotation()
	r.demote("foo")
	r.demote("bar")
	// Now demote foo again — it should move to the end
	r.demote("foo")
	// bar was demoted earlier now, so bar is preferred over foo
	assert.True(t, r.less("bar", "foo"))
	assert.False(t, r.less("foo", "bar"))
}

func TestNonDemotedPreserveOrder(t *testing.T) {
	r := newProxyRotation()
	// Neither demoted — less returns false (preserves original order)
	assert.False(t, r.less("foo", "bar"))
	assert.False(t, r.less("bar", "foo"))
}

func TestSingleProxyDemotion(t *testing.T) {
	r := newProxyRotation()
	r.demote("only")
	// A non-demoted host is preferred
	assert.True(t, r.less("other", "only"))
	// Demoting again doesn't panic
	r.demote("only")
	assert.True(t, r.less("other", "only"))
}
