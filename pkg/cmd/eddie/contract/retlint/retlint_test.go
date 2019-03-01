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

package retlint

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/cockroachdb/cockroach/pkg/cmd/eddie/ext"
	"github.com/cockroachdb/cockroach/pkg/cmd/eddie/rt"
	"github.com/cockroachdb/cockroach/pkg/util/syncutil"
	"github.com/stretchr/testify/assert"
)

type aggregator struct {
	fakePkgName string
	mu          struct {
		syncutil.Mutex
		dirty []dirtyFunction
	}
}

func (a *aggregator) Enforce(ctx ext.Context) error {
	linter := &RetLint{
		AllowedNames: []string{
			"_" + a.fakePkgName + "/GoodPtrError",
			"_" + a.fakePkgName + "/GoodValError",
		},
		TargetName: "error",
	}
	err := linter.Enforce(ctx)
	a.mu.Lock()
	a.mu.dirty = append(a.mu.dirty, linter.reported...)
	a.mu.Unlock()
	return err
}

var _ ext.Contract = &aggregator{}

func Test(t *testing.T) {
	a := assert.New(t)
	fakePkgName, err := filepath.Abs("./testdata")
	if !a.NoError(err) {
		return
	}

	aggregator := &aggregator{fakePkgName: fakePkgName}

	e := rt.Enforcer{
		Contracts: ext.ContractProviders{
			"RetLint": {New: func() ext.Contract { return aggregator }},
		},
		Dir:      fakePkgName,
		Logger:   log.New(os.Stdout, "", 0),
		Name:     "test",
		Packages: []string{"."},
	}

	_, err = e.Execute(context.Background())
	if !a.NoError(err) {
		return
	}
	//	a.Len(reports, 1)

	tcs := []struct {
		name      string
		whyLength int
	}{
		{name: "(*BadError).Self", whyLength: 1},
		{name: "DirectBad", whyLength: 2},
		{name: "DirectTupleBad", whyLength: 2},
		{name: "DirectTupleBadCaller", whyLength: 3},
		{name: "DirectTupleBadChain", whyLength: 3},
		{name: "ExplicitReturnVarBad", whyLength: 3},
		{name: "ExplicitReturnVarPhiBad", whyLength: 3},
		{name: "MakesIndirectCall", whyLength: 1},
		{name: "MakesInterfaceCallBad", whyLength: 2},
		{name: "PhiBad", whyLength: 3},
		{name: "ShortestWhyPath", whyLength: 1},
		{name: "TodoNoTypeInference", whyLength: 1}, // See doc on fn
		{name: "UsesSelfBad", whyLength: 2},
	}

	dirty := aggregator.mu.dirty
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
