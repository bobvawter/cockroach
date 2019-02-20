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
	"path/filepath"
	"regexp"
	"sort"

	"golang.org/x/tools/go/packages"
)

// Example:  contract:SomeContract {....}
var commentSyntax = regexp.MustCompile(`(?m)^//[[:space:]]*contract:([[:alnum:]]+)(.*)$`)

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
	pos := t.pkg.Fset.Position(t.Pos())
	return fmt.Sprintf("%s:%d:%d %s %s",
		filepath.Base(pos.Filename), pos.Line, pos.Column,
		t.contract, t.config)
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
	pos := a.pkg.Fset.Position(a.Pos())
	ptr := ""
	if a.ptr {
		ptr = "&"
	}
	return fmt.Sprintf("%s:%d:%d: var _ %s = %s%s{}",
		filepath.Base(pos.Filename), pos.Line, pos.Column,
		a.intf.Obj().Id(), ptr, a.str.Obj().Id())
}

var (
	_ sort.Interface = assertions{}
	_ sort.Interface = posses{}
	_ sort.Interface = targets{}
)

type assertions []*assertion

func (a assertions) Len() int           { return len(a) }
func (a assertions) Less(i, j int) bool { return a[i].Pos() < a[j].Pos() }
func (a assertions) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }

// What's the plural form of Pos?
type posses []token.Pos

func (p posses) Len() int           { return len(p) }
func (p posses) Less(i, j int) bool { return p[i] < p[j] }
func (p posses) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }

type targets []*target

func (t targets) Len() int           { return len(t) }
func (t targets) Less(i, j int) bool { return t[i].Pos() < t[j].Pos() }
func (t targets) Swap(i, j int)      { t[i], t[j] = t[j], t[i] }
