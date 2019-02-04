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
		name  string
		state state
	}{
		{"DirectGood", stateClean},
		{"PhiGood", stateClean},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			a := assert.New(t)

			for fn, stat := range l.stats {
				if fn.Name() == tc.name {
					a.Equalf(tc.state, stat.state, "was %s", stat.state)
					return
				}
			}
			a.Failf("did not find function named %q", tc.name)
		})
	}
}
