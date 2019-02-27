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
	"go/types"
	"log"
	"os"
	"testing"

	"github.com/cockroachdb/cockroach/pkg/cmd/eddie/ext"
	"github.com/stretchr/testify/assert"
	"golang.org/x/tools/go/ssa"
)

type checkKey struct {
	contract string
	name     string
	kind     ext.Kind
}
type check func(a *assert.Assertions, ctx ext.Context, r *recorder)

type cases map[checkKey]check

type recorder struct {
	t     *testing.T
	cases cases
	// To look like a MustReturnInt to the json decoder.
	Expected int
}

func (r *recorder) Enforce(ctx ext.Context) {
	key := checkKey{contract: ctx.Contract(), name: ctx.Name(), kind: ctx.Kind()}
	r.t.Run(fmt.Sprint(key), func(t *testing.T) {
		a := assert.New(t)
		fn := r.cases[key]
		if a.NotNilf(fn, "missing check %#v", key) {
			fn(a, ctx, r)
		}
	})
}

// This test creates a statically-configured Enforcer using the demo package.
func Test(t *testing.T) {
	a := assert.New(t)

	// Test cases are selected by the values return from
	// ext.Context.Contract(), Name(), and Kind().
	tcs := cases{
		{
			contract: "CanGoHere",
			name:     "ReturnOne",
			kind:     ext.KindInterfaceMethod,
		}: func(a *assert.Assertions, ctx ext.Context, _ *recorder) {
			// Verify that we see the declaring interface.
			a.IsType(&types.Interface{}, ctx.Declaration().(*ssa.Type).Type().Underlying())

			// Verify that we see the two implementing methods.
			a.Len(ctx.Objects(), 2)
			for _, obj := range ctx.Objects() {
				fn := obj.(*ssa.Function)
				a.Contains([]string{"ShouldPass", "ShouldFail"},
					fn.Signature.Recv().Type().(*types.Named).Obj().Name())
			}
		},

		{
			contract: "MustReturnInt",
			name:     "ReturnOne",
			kind:     ext.KindInterfaceMethod,
		}: func(a *assert.Assertions, ctx ext.Context, r *recorder) {
			// Verify that configuration actually happened; otherwise as above.
			a.Equal(1, r.Expected)
		},
	}

	newRecorder := func() ext.Contract { return &recorder{t, tcs, -1} }

	e := &Enforcer{
		Contracts: map[string]func() ext.Contract{
			"CanGoHere":     newRecorder,
			"MustReturnInt": newRecorder,
		},
		Dir:      "../demo",
		Logger:   log.New(os.Stdout, "", 0),
		Packages: []string{"."},
		Tests:    true,
	}

	if !a.NoError(e.execute(context.Background())) {
		return
	}

	a.Len(e.aliases, 1)
	a.Len(e.assertions, 4)
	a.Equal(len(e.targets), len(tcs), "target / test-case mismatch")

	// Call our recording contracts.
	a.NoError(e.enforceAll(context.Background()))

	// Check the target kinds.
	//seenKinds := make(map[ext.Kind]int)
	//for _, ctx := range seen {
	//	seenKinds[ctx.Kind()]++
	//}
	//
	//a.Equal(map[ext.Kind]int{
	//	ext.KindMethod:          1,
	//	ext.KindFunction:        1,
	//	ext.KindInterface:       1,
	//	ext.KindInterfaceMethod: 2,
	//	ext.KindType:            2,
	//}, seenKinds)
}
