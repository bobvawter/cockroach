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
		{name: "EnsureGoodValWithCommaOk", state: stateClean},
		{name: "EnsureGoodValWithSwitch", state: stateClean},
		{name: "EnsureGoodValWithTest", state: stateClean},
		{name: "MakesIndirectCall", state: stateDirty, whyLength: 1},
		{name: "PhiBad", state: stateDirty, whyLength: 3},
		{name: "PhiGood", state: stateClean},
		{name: "ShortestWhyPath", state: stateDirty, whyLength: 1},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			a := assert.New(t)

			for fn, stat := range l.stats {
				if fn.Name() == tc.name {
					a.Equalf(tc.state, stat.state, "was %s\n%s", stat.state, stat.stringify(l.pgm.Fset))
					a.Equalf(tc.whyLength, len(stat.why), "unexpected why length: %v", stat.why)
					return
				}
			}
			a.Fail("did not find function", tc.name)
		})
	}
}
