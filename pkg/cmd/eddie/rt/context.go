// Copyright 2019 The Cockroach Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or
// implied. See the License for the specific language governing
// permissions and limitations under the License.

package rt

import (
	"context"

	"github.com/cockroachdb/cockroach/pkg/cmd/eddie/ext"
	"github.com/cockroachdb/cockroach/pkg/util/syncutil"
	"golang.org/x/tools/go/ssa"
)

type contextImpl struct {
	context.Context

	declaration ssa.Member
	objects     []ssa.Member
	oracle      *ext.TypeOracle
	program     *ssa.Program
	reporter    func(*Result)
	target      *target

	mu struct {
		syncutil.Mutex
		dirty ext.Reporter
	}
}

var _ ext.Context = &contextImpl{}

// Contract implements ext.Context.
func (c *contextImpl) Contract() string { return c.target.contract }

// Declarations implements ext.Context.
func (c *contextImpl) Declaration() ssa.Member { return c.declaration }

// Kind implements ext.Context.
func (c *contextImpl) Kind() ext.Kind { return c.target.kind }

// Objects implements ext.Context.
func (c *contextImpl) Objects() []ssa.Member {
	if o := c.objects; o != nil {
		return o
	}
	return []ssa.Member{c.declaration}
}

// Oracle implements ext.Context.
func (c *contextImpl) Oracle() *ext.TypeOracle { return c.oracle }

// Program implements ext.Context.
func (c *contextImpl) Program() *ssa.Program { return c.program }

// Reporter implements ext.Context.
func (c *contextImpl) Reporter() ext.Reporter {
	var ret ext.Reporter
	c.mu.Lock()
	if ret = c.mu.dirty; ret == nil {
		var result *Result
		result, ret = newResultReporter(c.target.fset, c.target.pos,
			c.target.contract, c.declaration.Pos())
		c.mu.dirty = ret
		c.reporter(result)
	}
	c.mu.Unlock()
	return ret
}
