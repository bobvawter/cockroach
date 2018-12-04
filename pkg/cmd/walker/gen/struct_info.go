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
	Name() string
	Mode() refMode
	isVisitable()
}

var (
	_ visitableType = &namedInterfaceType{}
	_ visitableType = &structInfo{}
	_ visitableType = &namedVisitableType{}
	_ visitableType = &pointerType{}
	_ visitableType = &sliceType{}
)

// namedVisitableType represents a named type definitions like:
//  type Foos []Foo
//  type OptFoo *Foo
type namedVisitableType struct {
	*types.Named
	Underlying visitableType
}

func (*namedVisitableType) isVisitable() {}

func (t *namedVisitableType) Name() string {
	return t.Obj().Name()
}

func (t *namedVisitableType) Mode() refMode {
	return t.Underlying.Mode()
}

type namedInterfaceType struct {
	*types.Named
	*types.Interface
	v *visitation
}

func (t *namedInterfaceType) Name() string {
	return t.Obj().Name()
}

func (t *namedInterfaceType) Mode() refMode {
	return byRef
}

func (*namedInterfaceType) isVisitable() {}

// pointerType is a pointer to a visitableType.
type pointerType struct {
	Elem visitableType
}

func (*pointerType) isVisitable() {}

func (t *pointerType) Name() string {
	return "*" + t.Elem.Name()
}

func (t *pointerType) Mode() refMode {
	return byRef
}

// sliceType is a slice of a visitableType.
type sliceType struct {
	Elem visitableType
}

func (*sliceType) isVisitable() {}

func (t *sliceType) Name() string {
	return "[]" + t.Elem.Name()
}
func (t *sliceType) Mode() refMode {
	return byRef
}

// structInfo contains extracted information about a traversable struct.
type structInfo struct {
	*types.Named
	*types.Struct
	fields     []*fieldInfo
	fieldsDone bool
	ImplMode   refMode
	v          *visitation
}

func (*structInfo) isVisitable() {}

func (i *structInfo) Name() string {
	return i.Obj().Name()
}

// Mode is for consistency with field.Mode().
func (i *structInfo) Mode() refMode {
	return byValue
}

// Fields extracts and memoizes per-field data.
func (i *structInfo) Fields() []*fieldInfo {
	if i.fieldsDone {
		return i.fields
	}
	i.fieldsDone = true
	i.fields = make([]*fieldInfo, 0, i.NumFields())

	for a, j := 0, i.NumFields(); a < j; a++ {
		f := i.Field(a)
		// Ignore un-exported fields.
		if !f.Exported() {
			continue
		}

		// Look up `field Something` to visitableType.
		if found, ok := i.v.visitableType(f.Type()); ok {
			i.fields = append(i.fields, &fieldInfo{
				Name:   f.Name(),
				Parent: i,
				Target: found,
			})
		}
	}

	return i.fields
}

type fieldInfo struct {
	Name string
	// The structInfo that contains this fieldInfo.
	Parent *structInfo
	// The contents of the field.
	Target visitableType
}
