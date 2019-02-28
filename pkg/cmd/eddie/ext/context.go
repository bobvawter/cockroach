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
	"context"
	"go/token"

	"golang.org/x/tools/go/ssa"
)

// A Located object is associated with an opaque source location.
// Most types from the ssa package will implement this interface,
// as do all of the underlying AST objects.
type Located interface{ Pos() token.Pos }

// Context defines the interface between a Contract and the supporting
// framework.
type Context interface {
	context.Context
	// The name of the contract.
	Contract() string
	// Declaration returns the object that the contract declaration is
	// defined on. See additional discussion on the Contract type.
	Declaration() ssa.Member
	// Kind returns the kind of contract to be enforced.
	Kind() Kind
	// Objects returns a collection of objects that a specific contract
	// declaration maps to. In general, this will contain at least one
	// element, the value returned from Declaration(). See additional
	// discussion on the Kind type.
	Objects() []ssa.Member
	// Oracle returns a reference to a shared TypeOracle for answering
	// questions about the program's typesystem.
	Oracle() *TypeOracle
	// Program returns the SSA Program object which is driving the
	// analysis.
	Program() *ssa.Program
	// Report adds an error message to the output that is associated
	// with the given object.
	Report(l Located, msg string)
	// Reportf is a printf-style variant of Report.
	Reportf(l Located, msg string, args ...interface{})
}

// A Contract implements some correctness-checking logic.
type Contract interface {
	// Enforce will be called on an instance of the Contract automatically
	// by the runtime. Any error returned by this method will be reported
	// against the declaration object.
	Enforce(ctx Context) error
}

//go:generate stringer -type Kind -trimprefix Kind

// The Kind of a contract is a representation of how and where the
// contract binding was declared.
type Kind int

// The various kinds will inform the contract implementation as to
// what values it can expect to receive from the various Context methods.
//  | Kind             | Context.Declaration()     | Context.Objects()                 |
//  ------------------------------------------------------------------------------------
//  | Method           | *ssa.Function             | { Declaration() }                 |
//  | Function         | *ssa.Function             | { Declaration() }                 |
//  | Interface        | *ssa.Type (the interface) | []*ssa.Type (implementations)     |
//  | InterfaceMethod  | *ssa.Type (the interface) | []*ssa.Function (implementations) |
//  | Type             | *ssa.Type                 | { Declaration() }                 |
const (
	// A method declaration like:
	//   func (r Receiver) Foo() { ... }
	// presents just the function.
	KindMethod Kind = iota + 1
	// A top-level function declaration:
	//   func Foo () { .... }
	// presents just the function.
	KindFunction
	// An interface declaration:
	//   type I interface { ... }
	// presents the interface and all types which are known to
	// implement it.
	KindInterface
	// A method defined in an interface:
	//   type I interface { Foo() }
	// presents the interface method declaration and all functions which
	// are known to implement it.
	KindInterfaceMethod
	// All types other than interface declarations:
	//   type Foo int
	// presents the type declaration.
	KindType
)
