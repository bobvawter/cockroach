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

// Package retlint contains a linter which will perform a type-flow
// analysis on the concrete types returned from the methods defined
// in one or more packages.
package retlint

import (
	"go/types"

	"golang.org/x/tools/go/ssa"
)

type tightened struct {
	cached []types.Type
	calc   func() []types.Type
	parent *tightener
}

func (t *tightened) concrete() []types.Type {
	if t.calc != nil {
		t.cached = t.calc()
		t.calc = nil
	}
	return t.cached
}

type tightener struct {
	funcData map[*ssa.Function]*tightened
	// target contains the interface that we'll perform type-tightening for.
	target  *types.Interface
	valData map[ssa.Value]*tightened
}

func new() *tightener {
	return &tightener{
		funcData: make(map[*ssa.Function]*tightened),
		valData:  make(map[ssa.Value]*tightened),
	}
}

func (t *tightener) function(fn *ssa.Function) *tightened {
	if found, ok := t.funcData[fn]; ok {
		return found
	}
	ret := &tightened{
		calc: func() []types.Type {
			used := make(map[types.Type]bool)

			// Look for all return instructions and then work backwards.
			for _, block := range fn.Blocks {
				for _, inst := range block.Instrs {
					if retInst, ok := inst.(*ssa.Return); ok {
						for _, retVal := range retInst.Results {
							for _, retConcrete := range t.tighten(retVal).concrete() {
								used[retConcrete] = true
							}
						}
					}
				}
			}

			ret := make([]types.Type, 0, len(used))
			for typ := range used {
				ret = append(ret, typ)
			}
			return ret
		},
	}
	t.funcData[fn] = ret
	return ret
}

func (t *tightener) tighten(val ssa.Value) *tightened {
	return &tightened{
		calc: func() []types.Type {
			switch val := val.(type) {
			//			case *ssa.Call:
			default:
				return []types.Type{val.Type()}
			}
		},
	}
	//	typ := val.Type()
	//	if !types.AssertableTo(t.target, typ) {
	//		return empty
	//	}
	//	if found := t.valData[val]; found != nil {
	//		return found
	//	}
	//	data := &valueData{}
	//	t.valData[val] = data
	//	t.traceInto(val, data)
	//	return data
	//}
	//
	//func (t *tightener) traceInto(val ssa.Value, data *valueData) {
	//	data.static = val.Type()
	//
	//	switch tVal := val.(type) {
	//	case *ssa.Call:
	//		// Function call: Inherit from return variables
	//		if res := tVal.Common().Signature().Results(); res != nil {
	//			data.inheritsFrom = append(data.inheritsFrom, t.trace())
	//		}
	//	case *ssa.Extract:
	//		// We're accessing a member of a tuple, so we substitute the
	//		// nth value of the traced type data.
	//		actual := t.trace(tVal.Tuple).inheritsFrom[tVal.Index]
	//		data.inheritsFrom = []*valueData{actual}
	//	case *ssa.Phi:
	//		// A Phi ("phony") value represents the convergence of multiple
	//		// flows after a branch.  For example:
	//		//   var a Foo
	//		//   if condition {
	//		//     a = someFunc()
	//		//   } else {
	//		//     a = otherFunc()
	//		//   }
	//		//   doSomethingWith(a)
	//		//
	//		// The SSA of the above might look something like:
	//		//   Call(doSomethingWith, Phi(Call(someFunc), Call(otherFunc)))
	//		for _, edge := range tVal.Edges {
	//			data.inheritsFrom = append(data.inheritsFrom, t.trace(edge))
	//		}
	//	}
}
