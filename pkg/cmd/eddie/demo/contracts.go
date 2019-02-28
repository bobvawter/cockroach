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

var (
	_ ext.Contract = &CanGoHere{}
	_ ext.Contract = MustReturnInt{}
)

// MustReturnInt is an example of a trivial, but configurable, contract.
type MustReturnInt struct {
	Expected int64
}

// Enforce would be called twice in this example. Once for
// (ShouldPass).ReturnOne() and again for (ShouldFail).ReturnOne().
func (m MustReturnInt) Enforce(ctx ext.Context) error {
	for _, obj := range ctx.Objects() {
		fn, ok := obj.(*ssa.Function)
		if !ok {
			ctx.Report(obj, "is not a function")
			return nil
		}

		for _, block := range fn.Blocks {
			for _, inst := range block.Instrs {
				switch t := inst.(type) {
				case *ssa.Return:
					res := t.Results
					if len(res) != 1 {
						ctx.Report(t, "exactly one return value is required")
						return nil
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
	return nil
}

// CanGoHere is a no-op Contract which exists for documentation
// and testing purposes.
type CanGoHere struct{}

// Enforce implements the Contract interface and is a no-op.
func (*CanGoHere) Enforce(ctx ext.Context) error { return nil }
