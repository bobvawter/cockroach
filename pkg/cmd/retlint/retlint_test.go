package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoad(t *testing.T) {
	a := assert.New(t)
	l := RetLint{
		AllowedNames: []string{
			"testdata/GoodPtrError",
			"testdata/GoodValError",
		},
		Dir:        "./testdata",
		Packages:   []string{"."},
		TargetName: "error",
	}

	a.NoError(l.Execute())

	tcs := []struct {
		name      string
		state     state
		whyLength int
	}{
		{name: "DirectBad", state: stateDirty, whyLength: 2},
		{name: "DirectGood", state: stateClean},
		{name: "DirectTupleBad", state: stateDirty, whyLength: 2},
		{name: "DirectTupleBadChain", state: stateDirty, whyLength: 3},
		{name: "PhiBad", state: stateDirty, whyLength: 3},
		{name: "PhiGood", state: stateClean},
		{name: "ShortestWhyPath", state: stateDirty, whyLength: 1},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			a := assert.New(t)

			for fn, stat := range l.stats {
				if fn.Name() == tc.name {
					a.Equalf(tc.state, stat.state, "was %s", stat.state)
					a.Equalf(tc.whyLength, len(stat.why), "unexpected why length: %v", stat.why)
					return
				}
			}
			a.Fail("did not find function", tc.name)
		})
	}
}
