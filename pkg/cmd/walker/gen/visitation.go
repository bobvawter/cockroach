// Copyright 2018 The Cockroach Authors.
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
// permissions and limitations under the License. See the AUTHORS file
// for names of contributors.

package gen

import (
	"go/types"
)

// visitation encapsulates the state of generating a single
// visitable interface. This type is used extensively by the
// API template and exposes many convenience functions to keep
// the template simple.
type visitation struct {
	gen      *generation
	intf     *types.Interface
	intfName string
	inTest   bool
	Intfs    map[string]*namedInterfaceType
	pkg      *types.Package
	Structs  map[string]*structInfo
}

// populateGeneratedTypes finds top-level types that we will generate
// additional methods for.
func (v *visitation) populateGeneratedTypes() {
	g := v.gen
	scope := g.pkg.Scope()

	for _, name := range scope.Names() {
		obj := scope.Lookup(name)
		// Ignore un-exported types.
		if !obj.Exported() {
			continue
		}
		named, ok := obj.Type().(*types.Named)
		if !ok {
			continue
		}
		switch t := named.Underlying().(type) {
		case *types.Struct:
			var mode refMode
			if types.Implements(named, v.intf) {
				mode = byValue
			} else if types.Implements(types.NewPointer(named), v.intf) {
				mode = byRef
			} else {
				continue
			}
			v.Structs[name] = &structInfo{
				ImplMode: mode,
				Named:    named,
				Struct:   t,
				v:        v,
			}

		case *types.Interface:
			if types.Implements(named, v.intf) {
				v.Intfs[name] = &namedInterfaceType{
					Named:     named,
					Interface: t,
					v:         v,
				}
			}
		}
	}
}

// visitableType extracts the type information that we care about
// from typ. This handles named and anonymous types that are visitable.
func (v *visitation) visitableType(typ types.Type) (visitableType, bool) {
	switch t := typ.(type) {
	case *types.Named:
		// Ignore un-exported types.
		if !t.Obj().Exported() {
			return nil, false
		}
		switch u := t.Underlying().(type) {
		case *types.Struct:
			if s, ok := v.Structs[t.Obj().Name()]; ok {
				return s, true
			}
		case *types.Interface:
			if i, ok := v.Intfs[t.Obj().Name()]; ok {
				return i, true
			}
		default:
			// Any other named visitable type: type Foos []Foo
			if under, ok := v.visitableType(u); ok {
				return &namedVisitableType{Named: t, Underlying: under}, true
			}
		}

	case *types.Pointer:
		if elem, ok := v.visitableType(t.Elem()); ok {
			return &pointerType{Elem: elem}, true
		}
	case *types.Slice:
		if elem, ok := v.visitableType(t.Elem()); ok {
			return &sliceType{Elem: elem}, true
		}
	}
	return nil, false
}

// String is for debugging use only.
func (v *visitation) String() string {
	return v.intfName
}
