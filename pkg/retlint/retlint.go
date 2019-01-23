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
	"sort"
	"strings"

	"golang.org/x/tools/go/ssa"
)

func dedup(ts []types.Type) []types.Type {
	if len(ts) > 1 {
		filtered := ts[:0]
		seen := make(map[types.Type]bool, len(ts))
		for _, t := range ts {
			if !seen[t] {
				seen[t] = true
				filtered = append(filtered, t)
			}
		}

		sort.Slice(filtered, func(i, j int) bool {
			return strings.Compare(ts[i].String(), ts[j].String()) < 0
		})
		ts = filtered
	}
	return ts
}

// tightened represents a memoized, deferrable type-tightening
// computation. We want to be able to defer these computations in order
// to support call-graph cycles.
type tightened interface {
	at(index int) []types.Type
	len() int
}

type tightenedScalar struct {
	cached []types.Type
	calc   func() []types.Type
}

func exactly(ts ...types.Type) tightened {
	return &tightenedScalar{cached: dedup(ts)}
}

func (t *tightenedScalar) at(index int) []types.Type {
	if index != 0 {
		panic("invalid index for scalar")
	}
	if t.calc != nil {
		temp := t.calc
		t.calc = nil
		t.cached = temp()
	}
	return t.cached
}

func (*tightenedScalar) len() int {
	return 1
}

// tightenedTuple represents a tuple of tightened types. This is used
// for the return values of functions.
type tightenedTuple struct {
	values []tightened
}

func (t *tightenedTuple) at(index int) []types.Type {
	target := t.values[index]
	var temp []types.Type
	for i := 0; i < target.len(); i++ {
		temp = append(temp, target.at(i)...)
	}
	return dedup(temp)
}

func (t *tightenedTuple) len() int {
	return len(t.values)
}

type tightenedMerge struct {
	cached []*[]types.Type
	values []tightened
}

func merge(ts ...tightened) tightened {
	maxLen := 0
	for _, t := range ts {
		l := t.len()
		if l > maxLen {
			maxLen = l
		}
	}
	return &tightenedMerge{
		cached: make([]*[]types.Type, maxLen),
		values: ts,
	}
}

func (t *tightenedMerge) at(index int) []types.Type {
	if cached := t.cached[index]; cached != nil {
		return *cached
	}
	var ret []types.Type
	t.cached[index] = &ret
	for _, v := range t.values {
		if index < v.len() {
			ret = append(ret, v.at(index)...)
		}
	}
	ret = dedup(ret)
	return ret
}

func (t *tightenedMerge) len() int {
	return len(t.cached)
}

type tightener struct {
	funcData map[*ssa.Function]tightened
	// target contains the interface that we'll perform type-tightening for.
	target  *types.Interface
	valData map[ssa.Value]tightened
}

func new() *tightener {
	return &tightener{
		funcData: make(map[*ssa.Function]tightened),
		valData:  make(map[ssa.Value]tightened),
	}
}

func (t *tightener) function(fn *ssa.Function) tightened {
	if found, ok := t.funcData[fn]; ok {
		return found
	}
	ret := &tightenedTuple{}
	t.funcData[fn] = ret

	matrix := make([][]tightened, fn.Signature.Results().Len())

	// Look for all return instructions and then work backwards.
	for _, block := range fn.Blocks {
		for _, inst := range block.Instrs {
			if retInst, ok := inst.(*ssa.Return); ok {
				for i, retVal := range retInst.Results {
					matrix[i] = append(matrix[i], t.tighten(retVal))
				}
			}
		}
	}

	merged := make([]tightened, len(matrix))
	for i := range merged {
		merged[i] = merge(matrix[i]...)
	}
	ret.values = merged
	return ret
}

func (t *tightener) tighten(val ssa.Value) tightened {
	switch val := val.(type) {
	case *ssa.Call:
		// Function calls:

		// If the function is known to be statically-dispatched,
		// then we'll delegate to the tightened version of that function.
		if callee := val.Call.StaticCallee(); callee != nil {
			return t.function(callee)
		}

		// Otherwise, for a virtual-dispatch, the best thing that we can do
		// is to use the signature. Conceivably, for an interface
		// invocation, we could find all implementors of the interface and
		// merge them together.
		results := val.Call.Signature().Results()
		temp := make([]tightened, results.Len())
		for i := 0; i < results.Len(); i++ {
			temp[i] = exactly(results.At(i).Type())
		}
		return merge(temp...)

	case *ssa.Extract:
		return t.tighten(val.Tuple).(*tightenedTuple).values[val.Index]

	case *ssa.MakeInterface:
		// Interfaces are replaced by the type being wrapped into an interface.
		return t.tighten(val.X)

	case *ssa.Phi:
		// A Phi ("phony") value represents the convergence of multiple
		// flows after a branch.  For example:
		//   var a Foo
		//   if condition {
		//     a = someFunc()
		//   } else {
		//     a = otherFunc()
		//   }
		//   doSomethingWith(a)
		//
		// The SSA of the above might look something like:
		//   Call(doSomethingWith, Phi(Call(someFunc), Call(otherFunc)))
		ret := make([]tightened, 0, len(val.Edges))
		for _, edge := range val.Edges {
			ret = append(ret, t.tighten(edge))
		}
		return merge(ret...)

	default:
		return exactly(val.Type())
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
	//	}
}
