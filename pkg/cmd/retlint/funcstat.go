package main

import (
	"fmt"
	"strings"

	"golang.org/x/tools/go/ssa"
)

//go:generate stringer -type state -trimprefix state

type state int

const (
	stateUnknown state = iota
	stateAnalyzing
	stateClean
	stateDirty
)

var clean = &funcStat{state: stateClean}

type funcStat struct {
	dirties       map[*funcStat]*ssa.Call
	fn            *ssa.Function
	returns       []*ssa.Return
	state         state
	targetIndexes []int
	// why contains the shortest-dirty-path
	why []DirtyReason
}

var _ DirtyFunction = &funcStat{}

// Fn implements DirtyFunction
func (s *funcStat) Fn() *ssa.Function {
	return s.fn
}

func (s *funcStat) String() string {
	if s == clean {
		return "<Clean>"
	}
	sb := &strings.Builder{}

	fset := s.fn.Prog.Fset
	fmt.Fprintf(sb, "%s: func %s",
		fset.Position(s.fn.Pos()), s.fn.RelString(s.fn.Pkg.Pkg))
	for _, reason := range s.why {
		fmt.Fprintf(sb, "\n  %s: %s: %s",
			fset.Position(reason.Value.Pos()),
			reason.Reason,
			reason.Value)
	}

	return sb.String()
}

// Why implements DirtyFunction.
func (s *funcStat) Why() []DirtyReason {
	return s.why
}
