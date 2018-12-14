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
	"go/ast"
	"go/build"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"
)

// generation represents a run of the code generator. The overall
// flow is broken up into various stages, which can be seen in
// Execute().
type generation struct {
	astFiles []*ast.File
	// Allows additional files to be added to the parse phase for testing.
	extraTestSource map[string][]byte
	fileSet         *token.FileSet
	// By default, we don't fully type-check the input. This can be
	// enabled for testing to validate generated code.
	fullCheck bool
	inputDir  string
	pkg       *types.Package
	// The sources being considered.
	source *build.Package
	// The keys are the requested type names.
	visitations map[string]*visitation
	writeCloser func(name string) (io.WriteCloser, error)
}

// newGeneration constructs a generation which will look for the
// named interface types in the given directory.
func newGeneration(dir string, typeNames []string) *generation {
	ret := &generation{
		fileSet:     token.NewFileSet(),
		inputDir:    dir,
		visitations: make(map[string]*visitation),
		writeCloser: func(name string) (io.WriteCloser, error) {
			return os.OpenFile(name, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
		},
	}
	for _, name := range typeNames {
		ret.visitations[name] = nil
	}
	return ret
}

// Execute runs the complete code-generation cycle.
func (g *generation) Execute() error {
	if err := g.importSources(); err != nil {
		return err
	}

	files := append(g.source.GoFiles, g.source.TestGoFiles...)
	if len(g.extraTestSource) > 0 {
		// Mix in extra sources.
		if err := g.addSource(g.extraTestSource); err != nil {
			return err
		}
		// Filter our input sources if an input file is being overridden.
		filtered := files[:0]
		for _, file := range files {
			if g.extraTestSource[file] == nil {
				filtered = append(filtered, file)
			}
		}
		files = filtered
	}

	if err := g.parseFiles(files); err != nil {
		return err
	}
	if err := g.typeCheck(); err != nil {
		return err
	}
	if err := g.findInputInterfaces(); err != nil {
		return err
	}

	var grp errgroup.Group
	for name, v := range g.visitations {
		name, v := name, v
		grp.Go(func() error {
			if v == nil {
				return errors.Errorf("could not find interface %q", name)
			}
			v.populateGeneratedTypes()
			if err := v.generateAPI(); err != nil {
				return errors.Wrap(err, v.intfName)
			}
			return nil
		})
	}
	return grp.Wait()
}

func (g *generation) addSource(source map[string][]byte) error {
	for name, data := range source {
		astFile, err := parser.ParseFile(g.fileSet, name, string(data), 0 /* Mode */)
		if err != nil {
			return err
		}
		g.astFiles = append(g.astFiles, astFile)
	}
	return nil
}

func (g *generation) importSources() error {
	ctx := build.Default
	// Don't re-import code that we've generated.
	ctx.BuildTags = append(ctx.BuildTags, "walkerAnalysis")

	pkg, err := ctx.ImportDir(g.inputDir, 0)
	if err != nil {
		return err
	}
	g.source = pkg
	return nil
}

func (g *generation) parseFiles(files []string) error {
	for _, path := range files {
		astFile, err := parser.ParseFile(g.fileSet, filepath.Join(g.inputDir, path), nil, 0 /* Mode */)
		if err != nil {
			return err
		}
		g.astFiles = append(g.astFiles, astFile)
	}
	return nil
}

// typeCheck will run the go type checker over the parsed imports.
// This method is lenient, unless g.fullCheck has been set.
// The leniency helps in cases where code in the package that
// we're parsing depends on code that may not yet be generated.
func (g *generation) typeCheck() error {
	cfg := &types.Config{
		Importer: importer.For("source", nil),
	}
	if !g.fullCheck {
		cfg.DisableUnusedImportCheck = true
		// Just drain errors from the checker.
		cfg.Error = func(err error) {}
		cfg.IgnoreFuncBodies = true
	}
	var err error
	g.pkg, err = cfg.Check(g.inputDir, g.fileSet, g.astFiles, nil /* info */)
	if err != nil && g.fullCheck {
		return err
	}
	return nil
}

// findInputInterfaces looks for the interfaces named by the user.
func (g *generation) findInputInterfaces() error {
	scope := g.pkg.Scope()

	for name := range g.visitations {
		obj := scope.Lookup(name)
		// Look for named interfaces.
		if named, ok := obj.(*types.TypeName); ok {
			if intf, ok := named.Type().Underlying().(*types.Interface); ok {
				v := &visitation{
					gen: g,
					intf: namedInterfaceType{
						Named:     obj.Type().(*types.Named),
						Interface: intf,
					},
					intfName: named.Name(),
					Intfs:    make(map[string]namedInterfaceType),
					inTest:   strings.HasSuffix(g.fileSet.Position(obj.Pos()).Filename, "_test.go"),
					impl: namedInterfaceType{
						synthetic: named.Name() + "Impl",
					},
					pkg:     obj.Pkg(),
					Slices:  make(map[string]namedSliceType),
					Structs: make(map[string]namedStruct),
				}
				v.intf.v = v
				v.impl.v = v
				g.visitations[named.Name()] = v
			}
		}
	}
	return nil
}
