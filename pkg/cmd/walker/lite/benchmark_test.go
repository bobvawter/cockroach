// Copyright 2018 The Cockroach Authors.
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
// permissions and limitations under the License. See the AUTHORS file
// for names of contributors.

package lite_test

import (
	"testing"

	l "github.com/cockroachdb/cockroach/pkg/cmd/walker/lite"
)

// BenchmarkNoop should demonstrate that amortized visitations are
// allocation-free.
func BenchmarkNoop(b *testing.B) {

	b.Run("useValuePtrs=true", func(b *testing.B) {
		x, _ := dummyContainer(true)
		bench(b, x)
	})
	b.Run("useValuePtrs=false", func(b *testing.B) {
		x, _ := dummyContainer(false)
		bench(b, x)
	})
}

func bench(b *testing.B, x *l.ContainerType) {
	b.Helper()
	b.ReportAllocs()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			if _, _, err := x.WalkTarget(
				func(ctx l.TargetContext, x l.TargetImpl) (ret l.TargetDecision) { return }); err != nil {
				b.Fatal(err)
			}
		}
	})
}
