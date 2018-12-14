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

import "go/types"

// refMode captures the notion of a by-reference or by-value
// receiver type.
type refMode int

const (
	byValue refMode = iota
	byRef
)

// visitableType represents a type that we can generate visitation logic
// around:
//	* a named struct which implements the visitable interface,
//		either by-reference or by-value
//	* a named interface which implements the visitable interface
//	* a pointer to a visitable type
//	* a slice of a visitable type
//	* a named visitable type; e.g. "type Foos []Foo"
//	* TODO: a channel of a visitable type?
type visitableType interface {
	// String must return a codegen-safe representation of the type.
	String() string
	Visitation() *visitation
}

var (
	_ visitableType = namedStruct{}
	_ visitableType = namedInterfaceType{}
	_ visitableType = namedVisitableType{}
	_ visitableType = pointerType{}
	_ visitableType = namedSliceType{}
)

// namedVisitableType represents a named type definitions like:
//  type Foos []Foo
//  type OptFoo *Foo
type namedVisitableType struct {
	*types.Named
	Underlying visitableType
}

func (t namedVisitableType) String() string {
	return t.Obj().Name()
}

func (t namedVisitableType) Visitation() *visitation {
	return t.Underlying.Visitation()
}

type namedInterfaceType struct {
	*types.Named
	*types.Interface
	synthetic string
	v         *visitation
}

func (t namedInterfaceType) String() string {
	if t.synthetic != "" {
		return t.synthetic
	}
	return t.Obj().Name()
}

func (t namedInterfaceType) Visitation() *visitation {
	return t.v
}

// pointerType is a pointer to a visitableType.
type pointerType struct {
	Elem visitableType
}

func (t pointerType) String() string {
	return "*" + t.Elem.String()
}

func (t pointerType) Visitation() *visitation {
	return t.Elem.Visitation()
}

// namedSliceType is a slice of a visitableType.
type namedSliceType struct {
	*types.Named
	Elem      visitableType
	Synthetic string
}

func (namedSliceType) isVisitable() {}

func (t namedSliceType) String() string {
	if t.Named == nil {
		return t.Synthetic
	}
	return t.Named.Obj().Name()
}

func (t namedSliceType) Visitation() *visitation {
	return t.Elem.Visitation()
}

type namedStruct struct {
	*types.Named
	*types.Struct
	implMode refMode
	v        *visitation
}

func (i namedStruct) String() string {
	return i.Obj().Name()
}

func (i namedStruct) Fields() []fieldInfo {
	ret := make([]fieldInfo, 0, i.NumFields())

	for a, j := 0, i.NumFields(); a < j; a++ {
		f := i.Field(a)
		// Ignore un-exported fields.
		if !f.Exported() {
			continue
		}

		// Look up `field Something` to visitableType.
		if found, ok := i.v.visitableType(f.Type()); ok {
			ret = append(ret, fieldInfo{
				Name:   f.Name(),
				Parent: &i,
				Target: found,
			})
		}
	}

	return ret
}

// OpaqueFields returns the names of any field in the struct
// that we don't support and which should be copied as-is
// when an instance of the struct is cloned.
func (i namedStruct) OpaqueFields() []string {
	ret := make([]string, 0, i.NumFields())

	for a, j := 0, i.NumFields(); a < j; a++ {
		f := i.Field(a)
		// Ignore un-exported fields.
		if !f.Exported() {
			ret = append(ret, f.Name())
			continue
		}

		// Look up `field Something` to visitableType.
		if _, ok := i.v.visitableType(f.Type()); !ok {
			ret = append(ret, f.Name())
		}
	}

	return ret
}

func (i namedStruct) Visitation() *visitation {
	return i.v
}

type fieldInfo struct {
	Name string
	// The structInfo that contains this fieldInfo.
	Parent *namedStruct
	// The contents of the field.
	Target visitableType
}

// String returns the codegen-safe name of the field.
func (f fieldInfo) String() string {
	return f.Name
}
