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
	"go/token"
	"go/types"
	"path"
	"strings"

	"golang.org/x/tools/go/packages"
	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/ssautil"
)

func main() {

}

//go:generate stringer -type state -trimprefix state

type state int

const (
	stateUnknown state = iota
	stateAnalyzing
	stateClean
	stateDirty
)

type RetLint struct {
	AllowedNames []string
	Dir          string
	Packages     []string

	// The name of the target interface. This can be an unqualified name
	// like "error", which will be resolved against golang's "Universe"
	// scope, or something like "github.com/myproject/mypkg/SomeType".
	TargetName string

	// The acceptable types which implement the target interface.
	allowed map[*types.Named]bool
	pgm     *ssa.Program
	stats   map[*ssa.Function]*funcStat
	// The interfaces that we trigger the behavior on.
	target *types.Named
	work   []*funcStat
}

func (l *RetLint) Execute() error {
	l.allowed = make(map[*types.Named]bool)
	l.stats = make(map[*ssa.Function]*funcStat)

	cfg := &packages.Config{
		Dir:  l.Dir,
		Mode: packages.LoadAllSyntax,
	}
	pkgs, err := packages.Load(cfg, l.Packages...)
	if err != nil {
		return err
	}

	// Resolve the input types names.
	if found, err := resolve(pkgs, l.TargetName); err == nil {
		l.target = found
	} else {
		return err
	}

	for _, allowed := range l.AllowedNames {
		if found, err := resolve(pkgs, allowed); err == nil {
			l.allowed[found] = true
		} else {
			return err
		}
	}

	pgm, sPkgs := ssautil.AllPackages(pkgs, 0 /* Flags */)
	l.pgm = pgm
	pgm.Build()

	// Bootstrap the work to perform.
	for _, pkg := range sPkgs {
		for _, m := range pkg.Members {
			if fn, ok := m.(*ssa.Function); ok {
				l.stat(fn)
			}
		}
	}

	// Loop until we haven't added any new functions.
	for l.work != nil {
		work := l.work
		l.work = nil
		for _, stat := range work {
			l.analyze(stat)
		}
	}

	// Any functions not dirty by now are clean.
	for _, stat := range l.stats {
		if stat.state == stateAnalyzing {
			stat.state = stateClean
		}
	}
	return nil
}

func (l *RetLint) analyze(stat *funcStat) *funcStat {
	if stat.state == stateUnknown {
		stat.state = stateAnalyzing
		for _, ret := range stat.returns {
			for _, idx := range stat.targetIndexes {
				l.decide(stat, ret.Results[idx])

				if stat.state != stateAnalyzing {
					return stat
				}
			}
		}
	}
	return stat
}

// decide will mark the given function as dirty if the type of the given
// value is not statically-resolvable to one of the desired concrete types.
func (l *RetLint) decide(stat *funcStat, val ssa.Value) {
	switch t := val.(type) {
	case *ssa.Call:
		if callee := t.Call.StaticCallee(); callee != nil {
			next := l.analyze(l.stat(callee))
			switch next.state {
			case stateAnalyzing:
				next.dirties[stat] = true
			case stateDirty:
				why := append([]ssa.Value{t}, next.why...)
				l.markDirty(stat, why...)
			}
		} else {
			l.markDirty(stat, t)
		}

	case *ssa.Extract:
		l.decide(stat, t.Tuple)

	case *ssa.MakeInterface:
		// A value is being wrapped as an interface.
		l.decide(stat, t.X)

	case *ssa.Phi:
		// A Phi ("phony") value represents the convergence of multiple
		// flows after a branch.  For example:
		//   var a Foo
		//   if condition {
		//     a = someFunc()
		//   } else {
		//     a = otherFunc()
		//   }
		//   doSomethingWith(a)
		//
		// The SSA of the above might look something like:
		//   Call(doSomethingWith, Phi(Call(someFunc), Call(otherFunc)))
		for _, edge := range t.Edges {
			l.decide(stat, edge)
		}

	case *ssa.UnOp:
		// This is a dereference operation.
		if t.Op == token.MUL {
			l.decide(stat, t.X)
		}

	default:
		// Otherwise, see if the type is one of our named types or a pointer
		lookAt := t.Type()
		for {
			switch typ := lookAt.(type) {
			case *types.Pointer:
				lookAt = typ.Elem()
			case *types.Named:
				if !l.allowed[typ] {
					l.markDirty(stat, t)
				}
				return
			default:
				return
			}
		}
	}
}

// In the first pass, we'll extract all functions in the package.
func (l *RetLint) extract(fn *ssa.Function) {
	// Build is idempotent.
	fn.Package().Build()
	// Determine if the function returns a value of the target type.
	results := fn.Signature.Results()
	if results == nil {
		l.stats[fn] = clean
		return
	}

	var targetIndexes []int
	for i, j := 0, results.Len(); i < j; i++ {
		if named, ok := results.At(i).Type().(*types.Named); ok {
			if named == l.target {
				targetIndexes = append(targetIndexes, i)
			}
		}
	}
	if targetIndexes == nil {
		l.stats[fn] = clean
		return
	}

	// Extract all return statements from the function.
	var returns []*ssa.Return
	for _, block := range fn.Blocks {
		for _, inst := range block.Instrs {
			if ret, ok := inst.(*ssa.Return); ok {
				returns = append(returns, ret)
			}
		}
	}

	stat := l.stat(fn)
	stat.returns = returns
	stat.targetIndexes = targetIndexes
}

func (l *RetLint) markDirty(stat *funcStat, why ...ssa.Value) {
	// Try to choose a shorter explanation, if we can.
	if stat.why == nil || len(why) < len(stat.why) {
		stat.why = why
	}
	if stat.state == stateDirty {
		return
	}
	stat.state = stateDirty

	nextWhy := append(append(why, stat.why...))
	for chained := range stat.dirties {
		l.markDirty(chained, nextWhy...)
	}
}

func (l *RetLint) stat(fn *ssa.Function) *funcStat {
	ret := l.stats[fn]
	if ret == nil {
		ret = &funcStat{
			dirties: make(map[*funcStat]bool),
			fn:      fn,
		}
		l.stats[fn] = ret
		l.work = append(l.work, ret)
		l.extract(fn)
	}
	return ret
}

// resolve looks up a named type from within the collection of packages
func resolve(pkgs []*packages.Package, typeName string) (*types.Named, error) {
	tgtPath, tgtName := path.Split(typeName)
	var tgtObject types.Object
	if tgtPath == "" {
		tgtObject = types.Universe.Lookup(tgtName)
	} else {
		tgtPath = tgtPath[:len(tgtPath)-1]
		for _, pkg := range pkgs {
			if pkg.Name == tgtPath {
				tgtObject = pkg.Types.Scope().Lookup(tgtName)
				if tgtObject != nil {
					break
				}
			}
		}
	}
	if tgtObject == nil {
		return nil, fmt.Errorf("unable to find type %q", typeName)
	}
	if tgt, ok := tgtObject.Type().(*types.Named); ok {
		return tgt, nil
	} else {
		return nil, fmt.Errorf("%q was not a named type", tgtName)
	}
}

var clean = &funcStat{state: stateClean}

type funcStat struct {
	dirties       map[*funcStat]bool
	fn            *ssa.Function
	returns       []*ssa.Return
	state         state
	targetIndexes []int
	// why contains the shortest-dirty-path
	why []ssa.Value
}

func (s *funcStat) stringify(fs *token.FileSet) string {
	if s == clean {
		return "<Clean>"
	}
	sb := &strings.Builder{}

	pos := fs.Position(s.fn.Pos())
	fmt.Fprintf(sb, "%s: %s", pos, s.fn.Name())
	for _, reason := range s.why {
		fmt.Fprintf(sb, "\n  %s: %s", fs.Position(reason.Pos()), reason.String())
	}

	return sb.String()
}
