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
	"fmt"
	"go/token"

	"github.com/cockroachdb/cockroach/pkg/cmd/eddie/ext"
	"golang.org/x/tools/go/ssa"
)

type report struct {
	info string
	pos  token.Pos
}

type contextImpl struct {
	context.Context
	contract    string
	declaration ssa.Member
	kind        ext.Kind
	objects     []ssa.Member
	oracle      *ext.TypeOracle
	program     *ssa.Program
	reports     chan<- report
}

var _ ext.Context = &contextImpl{}

// Contract implements ext.Context.
func (c *contextImpl) Contract() string { return c.contract }

// Declarations implements ext.Context.
func (c *contextImpl) Declaration() ssa.Member { return c.declaration }

// Kind implements ext.Context.
func (c *contextImpl) Kind() ext.Kind { return c.kind }

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

// Declarations implements ext.Context.
func (c *contextImpl) Report(l ext.Located, msg string) {
	c.reports <- report{msg, l.Pos()}
}

// Reportf implements ext.Context.
func (c *contextImpl) Reportf(l ext.Located, msg string, args ...interface{}) {
	c.Report(l, fmt.Sprintf(msg, args...))
}
