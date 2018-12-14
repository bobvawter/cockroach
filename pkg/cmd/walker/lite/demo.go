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

// Package lite is used for demonstration and testing of walker.
package lite

//go:generate walker Target

// Target is a base interface that we run the code-generator against.
// There's nothing special about this interface.
type Target interface {
	Value() string
}

// Just an FYI to show that we support types that implement the
// interface by-value and by-reference.
var (
	_ Target = &ByRefType{}
	_ Target = ByValType{}
	_ Target = &ContainerType{}
	_ Target = &ignoredType{}
)

// EmbedsTarget demonstrates an interface hierarchy.
type EmbedsTarget interface {
	Target
	embedsTarget()
}

var (
	_ EmbedsTarget = ByValType{}
)

// Targets is a named slice of a visitable interface.
type Targets []Target

// ByRefType implements Target with a pointer receiver.
type ByRefType struct {
	Val string
}

// Value implements the Target interface.
func (x *ByRefType) Value() string { return x.Val }

// ContainerType is just a regular struct that contains fields
// whose types implement Target.
type ContainerType struct {
	ByRef         ByRefType
	ByRefPtr      *ByRefType
	ByRefSlice    []ByRefType
	ByRefPtrSlice []*ByRefType

	ByVal         ByValType
	ByValPtr      *ByValType
	ByValSlice    []ByValType
	ByValPtrSlice []*ByValType

	Container *ContainerType

	AnotherTarget    Target
	AnotherTargetPtr *Target

	// Interfaces which extend the visitable interface are supported.
	EmbedsTarget    EmbedsTarget
	EmbedsTargetPtr *EmbedsTarget

	// Slices of interfaces are supported.
	TargetSlice  []Target
	NamedTargets Targets

	// We can support slices of interface pointers.
	InterfacePtrSlice []*Target

	// Unexported fields aren't generated.
	ignored ByRefType
	// Unexported types aren't generated.
	Ignored *ignoredType
}

// Value implements the Target interface.
func (*ContainerType) Value() string { return "Container" }

// ByValType implements the Target interface with a value receiver.
type ByValType struct {
	Val string
}

func (ByValType) embedsTarget() {}

// Value implements the Target interface.
func (x ByValType) Value() string { return x.Val }

// ignoredType is not exported, so it won't appear in the API.
type ignoredType struct{}

// Value implements the Target interface.
func (ignoredType) Value() string { return "Should never see this" }
