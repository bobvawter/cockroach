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

package main_test

import (
	"path/filepath"
	"sort"
	"testing"

	retlint "github.com/cockroachdb/cockroach/pkg/cmd/retlint"
	"github.com/stretchr/testify/assert"
)

func Test(t *testing.T) {
	a := assert.New(t)
	fakePkgName, err := filepath.Abs("./testdata")
	if !a.NoError(err) {
		return
	}

	l := retlint.RetLint{
		AllowedNames: []string{
			"_" + fakePkgName + "/GoodPtrError",
			"_" + fakePkgName + "/GoodValError",
		},
		Dir:        fakePkgName,
		Packages:   []string{"."},
		TargetName: "error",
	}

	dirty, err := l.Execute()
	if !a.NoError(err) {
		return
	}

	tcs := []struct {
		name      string
		whyLength int
	}{
		{name: "(*BadError).Self", whyLength: 1},
		{name: "DirectBad", whyLength: 2},
		{name: "DirectTupleBad", whyLength: 2},
		{name: "DirectTupleBadCaller", whyLength: 3},
		{name: "DirectTupleBadChain", whyLength: 3},
		{name: "EnsureGoodValWithTest", whyLength: 1}, // XXX FIXME
		{name: "ExplicitReturnVarBad", whyLength: 3},
		{name: "ExplicitReturnVarPhiBad", whyLength: 3},
		{name: "MakesIndirectCall", whyLength: 1},
		{name: "PhiBad", whyLength: 3},
		{name: "ShortestWhyPath", whyLength: 1},
		{name: "UsesSelfBad", whyLength: 2},
	}

	t.Run("good extraction", func(t *testing.T) {
		a := assert.New(t)
		tcNames := make([]string, len(tcs))
		for i, tc := range tcs {
			tcNames[i] = tc.name
		}
		sort.Strings(tcNames)
		dirtyNames := make([]string, len(dirty))
		for i, d := range dirty {
			dirtyNames[i] = d.Fn().RelString(d.Fn().Pkg.Pkg)
		}
		sort.Strings(dirtyNames)
		a.Equal(tcNames, dirtyNames)
	})

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			a := assert.New(t)

			for _, d := range dirty {
				if d.Fn().RelString(d.Fn().Pkg.Pkg) == tc.name {
					a.Equalf(tc.whyLength, len(d.Why()), "unexpected why length:\n%s", d)
					return
				}
			}
			a.Fail("did not find function", tc.name)
		})
	}
}
