package main

import (
	"fmt"
	"go/token"
	"go/types"
	"path"

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

	_, sPkgs := ssautil.Packages(pkgs, ssa.LogSource|ssa.GlobalDebug)

	for _, pkg := range sPkgs {
		pkg.Build()
		for _, m := range pkg.Members {
			if fn, ok := m.(*ssa.Function); ok {
				l.extract(fn)
			}
		}
	}

	for l.work != nil {
		work := l.work
		l.work = nil
		for _, stat := range work {
			l.analyze(stat)
		}
	}

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
				l.markDirty(stat)
			}
		} else {
			l.markDirty(stat)
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
					l.markDirty(stat)
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
				break
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

func (l *RetLint) markDirty(stat *funcStat) {
	if stat.state == stateDirty {
		return
	}
	stat.state = stateDirty

	for other := range stat.dirties {
		l.markDirty(other)
	}
}

func (l *RetLint) stat(fn *ssa.Function) *funcStat {
	ret := l.stats[fn]
	if ret == nil {
		ret = &funcStat{}
		l.stats[fn] = ret
		l.work = append(l.work, ret)
	}
	return ret
}

// resolve looks up a named type from within the collection of packages
func resolve(pkgs []*packages.Package, typeName string) (*types.Named, error) {
	tgtPkg, tgtName := path.Split(typeName)
	var tgtObject types.Object
	if tgtPkg == "" {
		tgtObject = types.Universe.Lookup(tgtName)
	} else {
		tgtPkg = tgtPkg[:len(tgtPkg)-1]
		for _, pkg := range pkgs {
			if pkg.Name == tgtPkg {
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
	returns       []*ssa.Return
	state         state
	targetIndexes []int
}
