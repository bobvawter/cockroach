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

// In this test, we're going to show mutations performed in-place
// as well as mutations performed by replacement.  We have visitable
// types *ByRefType and ByValType.  We can modify *ByRefType in place,
// but must replace values of ByValType.

import (
	"strings"
	"testing"

	l "github.com/cockroachdb/cockroach/pkg/cmd/walker/lite"
	a "github.com/stretchr/testify/assert"
)

// Verify data extraction.
func TestChildAt(t *testing.T) {
	// Expect all but by-value values to be nil.
	t.Run("empty", func(t *testing.T) {
		assert := a.New(t)
		c := l.ContainerType{}
		for i, j := 0, c.NumChildren(); i < j; i++ {
			child := c.ChildAt(i)
			switch i {
			case 0, 4:
				assert.NotNilf(child, "at index %d", i)
			default:
				assert.Nilf(child, "at index %d", i)
			}
		}
	})

	// Only our inner *Container field should be nil.
	t.Run("useValuePtrs=true", func(t *testing.T) {
		assert := a.New(t)
		c, _ := dummyContainer(true)
		for i, j := 0, c.NumChildren(); i < j; i++ {
			child := c.ChildAt(i)
			switch i {
			case 8:
				assert.Nilf(child, "at index %d", i)
			default:
				assert.NotNilf(child, "at index %d", i)
			}
		}
	})
	t.Run("useValuePtrs=false", func(t *testing.T) {
		assert := a.New(t)
		c, _ := dummyContainer(false)
		for i, j := 0, c.NumChildren(); i < j; i++ {
			child := c.ChildAt(i)
			switch i {
			case 8:
				assert.Nilf(child, "at index %d", i)
			default:
				assert.NotNilf(child, "at index %d", i)
			}
		}
	})
}

// TestMutations applies a string-reversing visitor to our Container
// and then prints the resulting structure.
func TestMutations(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		checkMutations(t, &l.ContainerType{}, 0)
	})
	t.Run("useValuePtrs=true", func(t *testing.T) {
		x, count := dummyContainer(true)
		checkMutations(t, x, count)
	})
	t.Run("useValuePtrs=false", func(t *testing.T) {
		x, count := dummyContainer(false)
		checkMutations(t, x, count)
	})
}

func checkMutations(t *testing.T, x *l.ContainerType, count int) {
	t.Helper()
	assert := a.New(t)
	var expected string
	for i := 0; i < count; i++ {
		expected += "Hello"
	}

	x2, changed, err := x.WalkTarget(func(ctx l.TargetContext, x l.TargetImpl) (d l.TargetDecision) {
		switch t := x.(type) {
		case *l.ByRefType:
			cp := *t
			cp.Val = reverse(cp.Val)
			d = d.Replace(&cp)
		case *l.ByValType:
			cp := *t
			cp.Val = reverse(cp.Val)
			d = d.Replace(&cp)
		}
		return
	})
	if err != nil {
		t.Fatal(err)
	}
	assert.True(changed, "not changed")
	if x.ByRefPtr != nil {
		assert.NotEqual(x.ByRefPtr, x2.ByRefPtr, "pointer should have changed")
	}

	var w strings.Builder
	x3, changed, err := x2.WalkTarget(func(_ l.TargetContext, x l.TargetImpl) (d l.TargetDecision) {
		switch t := x.(type) {
		case *l.ByRefType:
			w.WriteString(t.Val)
		case *l.ByValType:
			w.WriteString(t.Val)
		}
		return
	})

	assert.Nil(err)
	assert.Equal(expected, w.String())
	assert.False(changed, "should not have changed")
	assert.Equal(x2.ByRefPtr, x3.ByRefPtr, "pointer should not have changed")
}

/*
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
*/

func dummyContainer(useValuePtrs bool) (*l.ContainerType, int) {
	count := 0
	olleh := func() string {
		count++
		return "olleH"
	}

	embedsTarget := func() l.EmbedsTarget {
		if useValuePtrs {
			return &l.ByValType{olleh()}
		} else {
			return l.ByValType{olleh()}
		}
	}

	target := func() l.Target {
		if useValuePtrs {
			return &l.ByValType{olleh()}
		} else {
			return l.ByValType{olleh()}
		}
	}

	p1 := target()
	p2 := target()
	p3 := target()
	p4 := embedsTarget()
	p5 := target()
	var nilTarget l.Target

	x := &l.ContainerType{
		ByRef:         l.ByRefType{olleh()},
		ByRefPtr:      &l.ByRefType{olleh()},
		ByRefSlice:    []l.ByRefType{{olleh()}, {olleh()}},
		ByRefPtrSlice: []*l.ByRefType{{olleh()}, nil, {olleh()}},

		ByVal:         l.ByValType{olleh()},
		ByValPtr:      &l.ByValType{olleh()},
		ByValSlice:    []l.ByValType{{olleh()}, {olleh()}},
		ByValPtrSlice: []*l.ByValType{{olleh()}, nil, {olleh()}},

		AnotherTarget:    target(),
		AnotherTargetPtr: &p5,

		EmbedsTarget:    &l.ByValType{olleh()},
		EmbedsTargetPtr: &p4,

		TargetSlice:  []l.Target{target(), target()},
		NamedTargets: []l.Target{target(), target()},

		InterfacePtrSlice: []*l.Target{&p1, nil, &nilTarget, &p2, &p3},
	}
	return x, count
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
