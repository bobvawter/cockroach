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

package rt

import (
	"context"
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"log"
	"os"
	"os/signal"
	"sort"
	"strings"
	"syscall"

	"github.com/cockroachdb/cockroach/pkg/cmd/eddie/ext"
	"github.com/cockroachdb/cockroach/pkg/util/syncutil"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
	"golang.org/x/tools/go/packages"
)

// Enforcer is the main entrypoint for a generated linter binary.
// The generated code will just configure an instance of Enforcer
// and call its Main() method.
type Enforcer struct {
	// Contracts contains providers for the various Contract types.
	// This map is the primary point of code-generation.
	Contracts map[string]func() ext.Contract
	// Allows the working directory to be overridden.
	Dir string
	// The name of the generated linter.
	Name string
	// An optional Logger to receive diagnostic messages.
	Logger *log.Logger
	// The package-patterns to enforce contracts upon.
	Packages []string
	// If true, the test sources for the package will be included.
	Tests bool

	aliases      targetAliases
	allPackages  map[string]*packages.Package
	assertions   []*assertion
	contractType *types.Interface
	pkgs         []*packages.Package
	targets      []*target
}

// Main is called by the generated main() code.
func (e *Enforcer) Main() {
	root := cobra.Command{
		Use:          e.Name,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			sig := make(chan os.Signal, 1)
			defer close(sig)

			signal.Notify(sig, syscall.SIGINT)
			defer signal.Stop(sig)

			go func() {
				select {
				case <-sig:
					e.println("Interrupted")
					cancel()
				}
			}()

			return e.execute(ctx)
		},
	}

	root.AddCommand(
		&cobra.Command{
			Use:   "contracts",
			Short: "Lists all defined contracts",
			Run: func(cmd *cobra.Command, _ []string) {
				for name := range e.Contracts {
					cmd.Println(name)
				}
			},
		})

	if err := root.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	os.Exit(0)
}

// execute is used by tests.
func (e *Enforcer) execute(ctx context.Context) error {
	// Load the source
	cfg := &packages.Config{
		Dir:   e.Dir,
		Fset:  token.NewFileSet(),
		Mode:  packages.LoadAllSyntax,
		Tests: e.Tests,
	}
	pkgs, err := packages.Load(cfg, e.Packages...)
	if err != nil {
		return err
	}

	e.allPackages = flattenImports(pkgs)

	// If the user has imported the ext package, they may have declared
	// contract aliases.  We'll need to find the underlying interface type.
	if extPkg := e.allPackages["github.com/cockroachdb/cockroach/pkg/cmd/eddie/ext"]; extPkg != nil {
		if obj := extPkg.Types.Scope().Lookup("Contract"); obj != nil {
			e.contractType = obj.Type().Underlying().(*types.Interface)
		}
	}

	e.pkgs = pkgs

	// Look for contract declarations on the AST side before we go through
	// the bother of converting to SSA form
	if err := e.findContracts(ctx); err != nil {
		return err
	}

	// Convert to SSA form.
	//	pgm, ssaPkgs := ssautil.AllPackages(pkgs, 0 /* mode */)

	// - Need to handle "forward-declared" contract aliases.
	// - Want to build up the datastructures that make the next pass easier

	// Aggregate contract declarations and resulting members.
	return nil
}

// findContracts performs AST-level extraction.  Specifically, it will
// find AST nodes which have been annotated with a contract declaration
// as well as type-assertion assignments.
func (e *Enforcer) findContracts(ctx context.Context) error {
	g, ctx := errgroup.WithContext(ctx)

	// mu protects the variables shared between goroutines.
	mu := struct {
		syncutil.Mutex
		aliases    targetAliases
		assertions assertions
		targets    targets
	}{
		aliases: make(targetAliases),
	}

	addAssertion := func(a *assertion) {
		e.println("assertion", a)
		mu.Lock()
		mu.assertions = append(mu.assertions, a)
		mu.Unlock()
	}

	// contract determine if the node has a contract comment.  If so, a
	// new target will be created.
	contract := func(pkg *packages.Package, comments []*ast.CommentGroup, node ast.Node, typ ast.Expr) {
		for _, group := range comments {
			for _, comment := range group.List {
				matches := commentSyntax.FindAllStringSubmatch(comment.Text, -1)
				for _, match := range matches {
					tgt := &target{
						config:   match[2],
						contract: match[1],
						node:     node,
						pkg:      pkg,
						pos:      comment.Pos(),
						typ:      pkg.TypesInfo.TypeOf(typ),
					}

					e.println("target", tgt)
					mu.Lock()
					// Special case for contract aliases of the form
					//   //contract:Foo { ... }
					//   type Alias ext.Contract
					if named, ok := tgt.typ.(*types.Named); ok && tgt.typ.Underlying() == e.contractType {
						name := named.Obj().Name()
						e.println("alias", name, ":=", tgt)
						mu.aliases[name] = append(mu.aliases[name], tgt)
					}
					mu.targets = append(mu.targets, tgt)
					mu.Unlock()
				}
			}
		}
	}

	// We want to resolve the ext.Contract interface object as we cycle
	// through the
	//	var contractIntf *types.Interface

	// Set up a parallel for-each loop over every input ast.File.
	for _, pkg := range e.pkgs {
		// See discussion on package.Config type for the naming scheme.
		if e.Tests && !strings.HasSuffix(pkg.ID, ".test]") {
			continue
		}

		for _, file := range pkg.Syntax {
			// Capture loop vars.
			pkg := pkg
			file := file
			g.Go(func() error {
				if ctx.Err() != nil {
					return nil
				}

				// CommentMap associates each node in the file with
				// its surrounding comments.
				comments := ast.NewCommentMap(pkg.Fset, file, file.Comments)

				// Now we'll inspect the ast.File and look for our magic
				// comment syntax.
				count := 0
				ast.Inspect(file, func(node ast.Node) bool {
					// We'll see a node==nil as the very last call.
					if node == nil {
						return false
					}
					// Occasionally check for cancellation.
					if count%1000 == 0 && ctx.Err() != nil {
						return false
					}
					count++

					switch t := node.(type) {
					case *ast.Field:
						// Fields of a function type, such as
						//   type I interface { func Blah() }
						//   type S struct { Blah func() }
						if funcType, ok := t.Type.(*ast.FuncType); ok {
							contract(pkg, comments[t], t, funcType)
						}
						return false

					case *ast.FuncDecl:
						// Top-level function or method declarations, such as
						//   func Foo() { .... }
						//   func (r Receiver) Bar() { ... }
						contract(pkg, comments[t], t, t.Type)
						// We don't need to descend into function bodies.
						return false

					case *ast.GenDecl:
						switch t.Tok {
						case token.TYPE:
							// Type declarations, such as
							//   type Foo struct { ... }
							//   type Bar interface { ... }
							for _, spec := range t.Specs {
								tSpec := spec.(*ast.TypeSpec)
								// Handle the usual case where contract is associated
								// with the type keyword.
								contract(pkg, comments[t], tSpec, tSpec.Name)
								// Handle unusual case where a type() block is being used
								// and a contract is specified on the entry.
								contract(pkg, comments[tSpec], tSpec, tSpec.Name)
								// We do need to descend into interfaces to pick up on
								// contracts applied only to interface methods.
								_, ok := tSpec.Type.(*ast.InterfaceType)
								return ok
							}

						case token.VAR:
							// Assertion declarations, such as
							//   var _ Intf = &Impl{}
							//   var _ Intf = Impl{}
							for _, spec := range t.Specs {
								v := spec.(*ast.ValueSpec)
								if len(v.Values) == 1 && v.Names[0].Name == "_" {
									if named, ok := pkg.TypesInfo.TypeOf(v.Type).(*types.Named); ok {
										if intf, ok := named.Underlying().(*types.Interface); ok {
											a := assertion{intf: named, pkg: pkg, pos: v.Pos()}
											switch v := pkg.TypesInfo.TypeOf(v.Values[0]).(type) {
											case *types.Named:
												if _, ok := v.Underlying().(*types.Struct); ok {
													a.str = v
												}
											case *types.Pointer:
												if named, ok := v.Elem().(*types.Named); ok {
													if _, ok := named.Underlying().(*types.Struct); ok {
														a.str = named
													}
												}
											}
											if a.str != nil {
												a.ptr = !types.Implements(a.str, intf)
												addAssertion(&a)
											}
										}
									}
								}
							}
						}
						return false
					default:
						return true
					}
				})
				return nil
			})
		}
	}

	if err := g.Wait(); err != nil {
		return err
	}

	// Produce stable output.
	for _, aliases := range mu.aliases {
		sort.Sort(aliases)
	}
	sort.Sort(mu.assertions)
	sort.Sort(mu.targets)

	e.aliases = mu.aliases
	e.assertions = mu.assertions
	e.targets = mu.targets
	return nil
}

// println will emit a diagnostic message via e.Logger, if one is configured.
func (e *Enforcer) println(args ...interface{}) {
	if l := e.Logger; l != nil {
		l.Println(args...)
	}
}
