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

package demo

import (
	"context"
	"strings"
	"testing"

	"github.com/pkg/errors"
)

// This test shows how existing logic that uses a single Pre/Post pair
// can be bridged to the walker API. This test will also cross-check
// the context stack exposed to users.
func TestStack(t *testing.T) {
	x := &ContainerType{
		ByRef: ByRefType{
			Val: "Hello",
		},
		ByRefPtr: &ByRefType{
			Val: "World!",
		},
		ByVal: ByValType{
			Val: "Hello",
		},
		ByValPtr: &ByValType{
			Val: "World!",
		},
		Container: &ContainerType{
			ByVal: ByValType{
				Val: "Hello",
			},
		},
	}
	const expected = `<root>
<root>.ByRef
<root>.ByRefPtr
<root>.ByVal
<root>.ByValPtr
<root>.Container
<root>.Container.ByRef
<root>.Container.ByVal
`

	depth := 0
	var w strings.Builder
	v := &TargetVisitorBase{
		DefaultPre: func(ctx TargetContext, x Target) (b bool, e error) {
			depth++

			s := ctx.Stack()
			if len(s) != ctx.StackLen() {
				return false, errors.Errorf("Stack depth mismatch: %d vs %d", len(s), ctx.StackLen())
			}

			w.WriteString(s.String())
			w.WriteRune('\n')

			if len(s) != depth {
				return false, errors.Errorf("expected depth %d, got %d", depth, len(s))
			}
			top := s[len(s)-1]
			match := false
			switch t := x.(type) {
			case *ContainerType:
				match = t == top.Value.(*ContainerType)
			case *ByRefType:
				match = t == top.Value.(*ByRefType)
			case ByValType:
				match = t == top.Value.(ByValType)
			}
			if !match {
				return false, errors.Errorf("did not see expected object: %+v vs %+v", x, top)
			}

			return true, nil
		},
		DefaultPost: func(ctx TargetContext, x Target) error {
			depth--
			return nil
		},
	}
	if _, _, err := x.Walk(context.Background(), v); err != nil {
		t.Fatal(err)
	}

	if w.String() != expected {
		t.Logf("Expected:\n%s\n\nGot:\n%s", expected, w.String())
	}
}

// Verify that errors returned by visitors are returned from Walk.
// We'll keep visiting a structure with an increasing event count
// until we see errors from both the pre and post handlers.
func TestErrorPropagation(t *testing.T) {
	x := &ContainerType{}

	var sawPre, sawPost bool
	for i := 0; i < 100; i++ {
		count := i
		_, _, err := x.Walk(context.Background(), TargetVisitorBase{
			DefaultPre: func(ctx TargetContext, x Target) (b bool, e error) {
				if count == 0 {
					return false, errors.Errorf("pre")
				}
				count--
				return true, nil
			},
			DefaultPost: func(ctx TargetContext, x Target) error {
				if count == 0 {
					return errors.Errorf("post")
				}
				count--
				return nil
			},
		})

		e := err.(*TargetWalkError)
		t.Log("Unwound", e.String())
		switch e.Error() {
		case "pre":
			sawPre = true
		case "post":
			sawPost = true
		default:
			t.Fatalf("unexpected error: %v", e)
		}
		if sawPre && sawPost {
			return
		}
	}
	t.Fatal("did not see expected events")
}
