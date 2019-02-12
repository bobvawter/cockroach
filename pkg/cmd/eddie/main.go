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
package main

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"reflect"

	"golang.org/x/tools/go/packages"
)

func main() {
}

// Eddie generates a contract-enforcer binary.  See discussion on the
// public API for details on the patterns that it looks for.
type Eddie struct {
	Dir      string
	Packages []string
}

func (e *Eddie) Execute() error {
	cfg := &packages.Config{
		Dir:  e.Dir,
		Mode: packages.LoadAllSyntax,
	}
	pkgs, err := packages.Load(cfg, e.Packages...)
	if err != nil {
		return err
	}

	// Look up the package name using reflection to prevent any weirdness
	// if the code gets moved to a new package.
	myType := reflect.TypeOf(Eddie{})
	lookFor := myType.PkgPath() + "/ext"

	// We're looking for some very specific patterns in the original
	// source code, so we're just going to rip through the AST nodes,
	// rather than try to back this out of the SSA form of the implicit
	// init() function.
	//
	// Specifically, we're looking for:
	// var _ Contract = MyContract{}
	// var _ Contract = &MyContract{}
	var contracts []types.Type
	for _, pkg := range pkgs {
		for _, f := range pkg.Syntax {
			for _, d := range f.Decls {
				if v, ok := d.(*ast.GenDecl); ok && v.Tok == token.VAR {
					for _, s := range v.Specs {
						if v, ok := s.(*ast.ValueSpec); ok &&
							len(v.Values) == 1 &&
							v.Names[0].Name == "_" {
							assignmentType, _ := pkg.TypesInfo.TypeOf(v.Type).(*types.Named)
							if assignmentType == nil ||
								assignmentType.Obj().Pkg().Path() != lookFor ||
								assignmentType.Obj().Name() != "Contract" {
								continue
							}
							valueType := pkg.TypesInfo.TypeOf(v.Values[0])
							contracts = append(contracts, valueType)
						}
					}
				}
			}
		}
	}

	fmt.Println(contracts)

	return nil
}
