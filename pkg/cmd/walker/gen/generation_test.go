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
	"io"
	"testing"

	"github.com/cockroachdb/cockroach/pkg/util/syncutil"
)

// Verify that our example data in the demo package is correct and
// that we won't break the existing test code with updated outputs.
// This test has two passes.  The first generates the code we want
// to emit and the second performs a complete type-checking of the
// demo package to make sure that any changes to the generated
// code will compile.
func TestExampleData(t *testing.T) {
	outputs := make(map[string][]byte)
	g := newGenerationForTesting("../demo", []string{"Target"}, outputs)

	err := g.Execute()
	for k, v := range outputs {
		t.Logf("%s\n%s\n\n\n", k, string(v))
	}
	if err != nil {
		t.Fatal(err)
	}

	if len(g.visitations) != 1 {
		t.Fatal("expecting 1 visitation")
	}
	v, ok := g.visitations["Target"]
	if !ok {
		t.Fatal("did not find Target interface")
	}
	if "Target" != v.intfName {
		t.Fatalf("wrong intfName %q", v.intfName)
	}
	if len(v.Structs) != 3 {
		t.Errorf("expecting 3 structs, got %d", len(v.Structs))
	}
	v.checkStructInfo(t, "ContainerType", byRef, 10)
	v.checkStructInfo(t, "ByValType", byValue, 0)
	v.checkStructInfo(t, "ByRefType", byRef, 0)

	v.checkVisitableInterface(t, "Target")
	v.checkVisitableInterface(t, "EmbedsTarget")

	g = newGenerationForTesting("../demo", []string{"Target"}, outputs)
	g.fullCheck = true
	g.extraTestSource = outputs
	if err := g.Execute(); err != nil {
		t.Fatal("could not parse with generated code", err)
	}
}

func (v *visitation) checkVisitableInterface(t *testing.T, name string) {
	t.Helper()
	obj := v.pkg.Scope().Lookup(name)
	if obj == nil {
		t.Errorf("no declaration of interface %s", name)
		return
	}
	vt, ok := v.visitableType(obj.Type())
	if !ok {
		t.Errorf("%s was not a visitableType", name)
		return
	}
	_, ok = vt.(*namedInterfaceType)
	if !ok {
		t.Errorf("%s was not a namedInterfaceType", name)
	}
}

func (v *visitation) checkStructInfo(t *testing.T, name string, implMode refMode, fieldCount int) {
	t.Helper()
	s, ok := v.Structs[name]
	if !ok {
		t.Errorf("did not find structInfo %s", name)
		return
	}
	if s.ImplMode != implMode {
		t.Errorf("%s implMode %v != %v", name, s.Mode(), implMode)
	}
	if len(s.Fields()) != fieldCount {
		t.Errorf("%s expected %d fields, got %d", s.Name(), len(s.Fields()), fieldCount)
	}
}

// newGenerationForTesting creates a generator that captures
// its output in the provided map.
func newGenerationForTesting(
	dir string, typeNames []string, outputs map[string][]byte,
) *generation {
	g := newGeneration(dir, typeNames)
	var mu syncutil.Mutex
	g.writeCloser = func(name string) (io.WriteCloser, error) {
		return newMapWriter(name, &mu, outputs), nil
	}
	return g
}
