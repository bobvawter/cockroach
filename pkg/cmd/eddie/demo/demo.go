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

package demo

import (
	"go/constant"

	"github.com/cockroachdb/cockroach/pkg/cmd/eddie/ext"
	"golang.org/x/tools/go/ssa"
)

// The framework will look for these kinds of type-assertion
// declarations when deciding whether or not some type implements the
// Contract interface or when looking for structs that implement an
// interface which participates in a contract.
var (
	_ ext.Contract  = MustReturnInt{}
	_ ReturnsNumber = ShouldPass{}
	_ ReturnsNumber = ShouldFail{}
)

// MustReturnInt is an example of a trivial, but configurable, contract.
type MustReturnInt struct {
	Expected int64
}

// Enforce would be called twice in this example. Once for
// (ShouldPass).ReturnOne() and again for (ShouldFail).ReturnOne().
func (m MustReturnInt) Enforce(ctx ext.Context) {
	for _, obj := range ctx.Objects() {
		fn, ok := obj.(*ssa.Function)
		if !ok {
			ctx.Report(obj, "is not a function")
			return
		}

		for _, block := range fn.Blocks {
			for _, inst := range block.Instrs {
				switch t := inst.(type) {
				case *ssa.Return:
					res := t.Results
					if len(res) != 1 {
						ctx.Report(t, "exactly one return value is required")
						return
					}
					if c, ok := res[0].(*ssa.Const); ok {
						if constant.MakeInt64(m.Expected) != c.Value {
							ctx.Reportf(c, "expecting %d, got %s", m.Expected, c.Value)
						}
					} else {
						ctx.Report(res[0], "not a constant value")
					}
				}
			}
		}
	}
}

// ReturnsNumber defines a contract on its only method.
type ReturnsNumber interface {
	// This is a normal doc-comment, except that it has a magic
	// comment below, consisting of a contract name and a
	// JSON block which will be unmarshalled into the contract
	// struct instance.
	//
	//contract:MustReturnInt { "Expected" : 1 }
	ReturnOne() int
}

type ShouldPass struct{}

func (ShouldPass) ReturnOne() int {
	return 1
}

type ShouldFail struct{}

func (ShouldFail) ReturnOne() int {
	return 0
}
