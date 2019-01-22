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

package retlint

import (
	"fmt"
	"go/types"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/tools/go/packages"
	"golang.org/x/tools/go/ssa/ssautil"
)

func TestTightener(t *testing.T) {
	a := assert.New(t)
	pkgs, err := packages.Load(&packages.Config{
		Dir:  "./testdata",
		Mode: packages.LoadSyntax,
	}, ".")

	if !a.NoError(err) {
		return
	}

	_, ssaPkgs := ssautil.Packages(pkgs, 0 /* default mode */)
	if !a.Len(ssaPkgs, 1) {
		return
	}
	pkg := ssaPkgs[0]
	pkg.Build()

	testData := []struct {
		name     string
		expected []string
	}{
		{"returnsConcrete", []string{"Concrete"}},
		{"returnsConcretePtr", []string{"Concrete"}},
		{"returnsConcreteAsTarget", []string{"Concrete"}},
		{"returnsConcretePtrAsTarget", []string{"Concrete"}},
		{"phiSimple", []string{"Bar", "Foo"}},
	}

	for _, td := range testData {
		t.Run(td.name, func(t *testing.T) {
			a := assert.New(t)
			x := new()
			fn := pkg.Func(td.name)
			if !a.NotNil(fn) {
				return
			}
			asMap := make(map[string]types.Type)
			for _, conc := range x.function(fn).concrete() {
			conc:
				for {
					switch tConc := conc.(type) {
					case *types.Named:
						asMap[tConc.Obj().Name()] = conc
						break conc
					case *types.Pointer:
						// Resolve *Foo to Foo
						conc = tConc.Elem()
					default:
						panic(fmt.Sprintf("unimplemented: %s", reflect.TypeOf(tConc)))
					}
				}
			}

			for _, ex := range td.expected {
				a.Contains(asMap, ex, "missing type")
			}
		})
	}
}
