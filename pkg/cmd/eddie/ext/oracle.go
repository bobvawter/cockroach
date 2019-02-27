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

package ext

import (
	"go/types"

	"github.com/cockroachdb/cockroach/pkg/util/syncutil"
	"golang.org/x/tools/go/ssa"
)

type Assertions map[types.Object][]types.Object

// A TypeOracle answers questions about a program's typesystem.
// All methods are safe to call from multiple goroutines.
type TypeOracle struct {
	assertedImplementors map[*types.Interface][]types.Object
	pgm                  *ssa.Program
	mu                   struct {
		syncutil.RWMutex
		typeImplementors map[*types.Interface][]types.Object
	}
}

// NewOracle constructs a TypeOracle.  In general, contract
// implementations should prefer the shared instance provided by
// Context, rather than constructing a new one.
func NewOracle(pgm *ssa.Program, assertions Assertions) *TypeOracle {
	ret := &TypeOracle{
		assertedImplementors: make(map[*types.Interface][]types.Object, len(assertions)),
		pgm:                  pgm,
	}
	for k, v := range assertions {
		if intf, ok := k.Type().Underlying().(*types.Interface); ok {
			ret.assertedImplementors[intf] = v
		}
	}
	ret.mu.typeImplementors = make(map[*types.Interface][]types.Object)
	return ret
}

// MethodImplementors finds the named method on all types which implement
// the given interface.
func (o *TypeOracle) MethodImplementors(
	intf *types.Interface, name string, assertedOnly bool,
) []*ssa.Function {
	impls := o.TypeImplementors(intf, assertedOnly)
	ret := make([]*ssa.Function, len(impls))
	for i, impl := range impls {
		ret[i] = o.pgm.LookupMethod(impl.Type(), impl.Pkg(), name)
	}
	return ret
}

// TypeImplementors returns the runtime times which implement the
// given interface, according to explicit assertions made by the user.
func (o *TypeOracle) TypeImplementors(intf *types.Interface, assertedOnly bool) []types.Object {
	var ret []types.Object

	if assertedOnly {
		ret = o.assertedImplementors[intf]
	} else {
		o.mu.RLock()
		// We may insert nil slices later on, so use comma-ok.
		ret, found := o.mu.typeImplementors[intf]
		o.mu.RUnlock()

		if !found {
			for _, typ := range o.pgm.RuntimeTypes() {
				var lastName types.Object
			chase:
				for {
					switch t := typ.(type) {
					case *types.Pointer:
						typ = t.Elem()
					case *types.Named:
						lastName = t.Obj()
						typ = t.Underlying()
					default:
						break chase
					}
				}
				if lastName != nil {
					ret = append(ret, lastName)
				}
			}

			o.mu.Lock()
			o.mu.typeImplementors[intf] = ret
			o.mu.Unlock()
		}
	}

	// Return copies of non-nil slices.
	if ret != nil {
		ret = append(ret[:0], ret...)
	}
	return ret
}
