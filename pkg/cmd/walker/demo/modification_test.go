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

// In this test, we're going to show mutations performed in-place
// as well as mutations performed by replacement.  We have visitable
// types *ByRefType and ByValType.  We can modify *ByRefType in place,
// but must replace values of ByValType.

import (
	"context"
	"strings"
	"testing"
)

type Printer struct {
	TargetVisitorBase
	w strings.Builder
}

var _ TargetVisitor = &Printer{}

func (p *Printer) PreByRefType(ctx TargetContext, foo *ByRefType) (bool, error) {
	p.w.WriteString(foo.Val)
	return false, nil
}

func (p *Printer) PreByValType(ctx TargetContext, x ByValType) (bool, error) {
	p.w.WriteString(x.Val)
	return false, nil
}

type Mutator struct {
	TargetVisitorBase
	badMutation bool
}

var _ TargetVisitor = &Mutator{}

// We're going to mutate ByRefTypes in-place.
func (Mutator) PostByRefType(ctx TargetContext, x *ByRefType) error {
	x.Val = reverse(x.Val)
	return nil
}

// We're going to replace ByValuetypes.
func (m Mutator) PostByValType(ctx TargetContext, x ByValType) error {
	if m.badMutation {
		ctx.Replace(&ByRefType{})
	}
	x.Val = reverse(x.Val)
	ctx.Replace(x)
	// Just to be explicit that once the replacement has happened,
	// it's all by-value.
	x.Val = "Should never see this."
	return nil
}

// TestMutations applies a string-reversing visitor to our Container
// and then prints the resulting structure.
func TestMutations(t *testing.T) {
	count := 0
	olleh := func() string {
		count++
		return "olleH"
	}

	p1 := Target(&ByRefType{olleh()})
	p2 := Target(nil)
	p3 := Target(ByValType{olleh()})
	p4 := EmbedsTarget(ByValType{olleh()})

	x := &ContainerType{
		AnonymousTargetSlice: []Target{
			&ByRefType{olleh()},
			nil,
			ByValType{olleh()},
		},
		ByRef:           ByRefType{olleh()},
		ByRefPtr:        &ByRefType{olleh()},
		ByVal:           ByValType{olleh()},
		ByValPtr:        &ByValType{olleh()},
		EmbedsTarget:    &ByValType{olleh()},
		EmbedsTargetPtr: &p4,
		InterfacePtrSlice: []*Target{
			&p1,
			nil,
			&p2,
			&p3,
		},
		NamedTargets: Targets{
			&ByRefType{olleh()},
			nil,
			ByValType{olleh()},
		},
	}
	var expected string
	for i := 0; i < count; i++ {
		expected += "Hello"
	}

	x2, changed, err := x.Walk(context.Background(), Mutator{})
	if err != nil {
		t.Fatal(err)
	}
	if !changed {
		t.Fatal("not changed")
	}
	if x.ByRefPtr != x2.ByRefPtr {
		t.Fatal("pointer should not have changed")
	}

	sv := &Printer{}
	x3, changed, err := x2.Walk(context.Background(), sv)
	if err != nil {
		t.Fatal(err)
	}

	if sv.w.String() != expected {
		t.Errorf("unexpected print result: %s", sv.w.String())
	}

	if changed {
		t.Fatal("should not have changed")
	}
	if x2.ByRefPtr != x3.ByRefPtr {
		t.Fatal("pointer should not have changed")
	}
}

func TestBadMutations(t *testing.T) {
	x := &ContainerType{
		ByRef: ByRefType{
			Val: "olleH",
		},
		ByRefPtr: &ByRefType{
			Val: "!dlroW",
		},
		ByVal: ByValType{
			Val: "olleh",
		},
		ByValPtr: &ByValType{
			Val: "!dlroW",
		},
	}
	_, _, err := x.Walk(context.Background(), Mutator{badMutation: true})
	switch tErr := err.(type) {
	case *TargetWalkError:
		if "*demo.ByRefType is not assignable to demo.ByValType at <root>.ByVal" != tErr.String() {
			t.Errorf("unexpected error string %q", tErr.String())
		}
	default:
		t.Fatal("unexpected error type")
	}
}

// Via Russ Cox
// https://groups.google.com/d/msg/golang-nuts/oPuBaYJ17t4/PCmhdAyrNVkJ
func reverse(s string) string {
	n := 0
	runes := make([]rune, len(s))
	for _, r := range s {
		runes[n] = r
		n++
	}
	// Account for multi-byte points.
	runes = runes[0:n]
	// Reverse.
	for i := 0; i < n/2; i++ {
		runes[i], runes[n-1-i] = runes[n-1-i], runes[i]
	}

	return string(runes)
}
