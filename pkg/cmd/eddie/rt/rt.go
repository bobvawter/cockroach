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

// Package rt contains the runtime code which will support a generated
// enforcer binary.
package rt

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"regexp"
	"sort"

	"golang.org/x/tools/go/packages"
)

var commentSyntax = regexp.MustCompile(`(?m)^//[\w]*contract:([\w]+)(.*)$`)

type target struct {
	config   string
	contract string
	node     ast.Node
	pkg      *packages.Package
	pos      token.Pos
	typ      types.Type
}

// Pos implement the Located interface.
func (t *target) Pos() token.Pos {
	return t.pos
}

// String is for debugging use only.
func (t *target) String() string {
	return fmt.Sprintf("%s %s %s",
		t.pkg.Fset.Position(t.Pos()), t.contract, t.config)
}

// An assertion represents a top-level declaration of the forms
//   var _ SomeInterface = SomeStruct{}
//   var _ SomeInterface = &SomeStruct{}
type assertion struct {
	// A named interface type.
	intf *types.Named
	pkg  *packages.Package
	pos  token.Pos
	// Indicates that the interface is implemented using pointer receivers.
	ptr bool
	// A named struct type.
	str *types.Named
}

// Pos implements the Located interface.
func (a *assertion) Pos() token.Pos {
	return a.pos
}

// String is for debugging use only.
func (a *assertion) String() string {
	var ptr = ""
	if a.ptr {
		ptr = "&"
	}
	return fmt.Sprintf("%s var _ %s = %s%s{}",
		a.pkg.Fset.Position(a.pos), a.intf.Obj().Id(), ptr, a.str.Obj().Id())
}

type posses []token.Pos

var _ sort.Interface = posses{}

func (p posses) Len() int           { return len(p) }
func (p posses) Less(i, j int) bool { return p[i] < p[j] }
func (p posses) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }
