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

package engine_test

import (
	"testing"
	"unsafe"

	"github.com/cockroachdb/cockroach/pkg/cmd/walker/engine"
)

type Demoable interface {
	demoable()
}

type Demo struct {
	Foo         Foo
	FooPtr      *Foo
	FooSlice    []Foo
	FooPtrSlice []*Foo

	Bar         Bar
	BarPtr      *Bar
	BarSlice    []Bar
	BarPtrSlice []*Bar

	// Demonstrate cycle-breaking.
	DemoPtr *Demo

	Demoable      Demoable
	DemoableSlice []Demoable
}

func (*Demo) demoable() {}

type Foo struct {
	Val string
}

func (*Foo) demoable() {}

type Bar struct {
	Val string
}

func (Bar) demoable() {}

const (
	_ engine.TypeId = iota
	TypeIdDemo
	TypeIdDemoPtr

	TypeIdFoo
	TypeIdFooPtr
	TypeIdFooSlice
	TypeIdFooPtrSlice

	TypeIdBar
	TypeIdBarPtr
	TypeIdBarSlice
	TypeIdBarPtrSlice

	TypeIdDemoable
	TypeIdDemoableSlice
)

var DemoMap = engine.TypeMap{
	TypeIdDemo: {
		Copy: func(dest, from unsafe.Pointer) {
			*(*Demo)(dest) = *(*Demo)(from)
		},
		Facade: func(ctx engine.Context, fn engine.FacadeFn, x unsafe.Pointer) engine.Decision {
			return fn.(WalkerFn)(ctx, (*Demo)(x))
		},
		Fields: []engine.FieldInfo{
			{"Foo", unsafe.Offsetof(Demo{}.Foo), TypeIdFoo},
			{"FooPtr", unsafe.Offsetof(Demo{}.FooPtr), TypeIdFooPtr},
			{"FooSlice", unsafe.Offsetof(Demo{}.FooSlice), TypeIdFooSlice},
			{"FooPtrSlice", unsafe.Offsetof(Demo{}.FooPtrSlice), TypeIdFooPtrSlice},
			{"Bar", unsafe.Offsetof(Demo{}.Bar), TypeIdBar},
			{"BarPtr", unsafe.Offsetof(Demo{}.BarPtr), TypeIdBarPtr},
			{"BarSlice", unsafe.Offsetof(Demo{}.BarSlice), TypeIdBarSlice},
			{"BarPtrSlice", unsafe.Offsetof(Demo{}.BarPtrSlice), TypeIdBarPtrSlice},
			{"DemoPtr", unsafe.Offsetof(Demo{}.DemoPtr), TypeIdDemoPtr},
			{"Demoable", unsafe.Offsetof(Demo{}.Demoable), TypeIdDemoable},
			{"DemoableSlice", unsafe.Offsetof(Demo{}.DemoableSlice), TypeIdDemoableSlice},
		},
		New:      func() unsafe.Pointer { return unsafe.Pointer(&Demo{}) },
		SizeOf:   unsafe.Sizeof(Demo{}),
		TypeKind: engine.KindStruct,
		TypeId:   TypeIdDemo,
	},
	TypeIdDemoPtr: {
		Copy: func(dest, from unsafe.Pointer) {
			*(**Demo)(dest) = *(**Demo)(from)
		},
		Elem:     TypeIdDemo,
		SizeOf:   unsafe.Sizeof(&Demo{}),
		TypeKind: engine.KindPointer,
		TypeId:   TypeIdDemoPtr,
	},
	TypeIdFoo: {
		Copy: func(dest, from unsafe.Pointer) {
			*(*Foo)(dest) = *(*Foo)(from)
		},
		Facade: func(ctx engine.Context, fn engine.FacadeFn, x unsafe.Pointer) engine.Decision {
			return fn.(WalkerFn)(ctx, (*Foo)(x))
		},
		New:      func() unsafe.Pointer { return unsafe.Pointer(&Foo{}) },
		SizeOf:   unsafe.Sizeof(Foo{}),
		TypeKind: engine.KindStruct,
		TypeId:   TypeIdFoo,
	},
	TypeIdFooPtr: {
		Copy: func(dest, from unsafe.Pointer) {
			*(**Foo)(dest) = *(**Foo)(from)
		},
		Elem:     TypeIdFoo,
		SizeOf:   unsafe.Sizeof(&Foo{}),
		TypeKind: engine.KindPointer,
		TypeId:   TypeIdFooPtr,
	},
	TypeIdFooSlice: {
		Copy: func(dest, from unsafe.Pointer) {
			*(*[]Foo)(dest) = *(*[]Foo)(from)
		},
		Elem: TypeIdFoo,
		NewSlice: func(size int) unsafe.Pointer {
			x := make([]Foo, size)
			return unsafe.Pointer(&x)
		},
		SizeOf:   unsafe.Sizeof([]Foo{}),
		TypeKind: engine.KindSlice,
		TypeId:   TypeIdFooSlice,
	},
	TypeIdFooPtrSlice: {
		Copy: func(dest, from unsafe.Pointer) {
			*(*[]*Foo)(dest) = *(*[]*Foo)(from)
		},
		NewSlice: func(size int) unsafe.Pointer {
			x := make([]*Foo, size)
			return unsafe.Pointer(&x)
		},
		SizeOf:   unsafe.Sizeof([]*Foo{}),
		TypeKind: engine.KindSlice,
		Elem:     TypeIdFooPtr,
		TypeId:   TypeIdFooPtrSlice,
	},
	TypeIdBar: {
		Copy: func(dest, from unsafe.Pointer) {
			*(*Bar)(dest) = *(*Bar)(from)
		},
		Facade: func(ctx engine.Context, fn engine.FacadeFn, x unsafe.Pointer) engine.Decision {
			return fn.(WalkerFn)(ctx, (*Bar)(x))
		},
		New:      func() unsafe.Pointer { return unsafe.Pointer(&Bar{}) },
		SizeOf:   unsafe.Sizeof(Bar{}),
		TypeKind: engine.KindStruct,
		TypeId:   TypeIdBar,
	},
	TypeIdBarPtr: {
		Copy: func(dest, from unsafe.Pointer) {
			*(**Bar)(dest) = *(**Bar)(from)
		},
		Elem:     TypeIdBar,
		SizeOf:   unsafe.Sizeof(&Bar{}),
		TypeKind: engine.KindPointer,
		TypeId:   TypeIdBarPtr,
	},
	TypeIdBarSlice: {
		Copy: func(dest, from unsafe.Pointer) {
			*(*[]Bar)(dest) = *(*[]Bar)(from)
		},
		NewSlice: func(size int) unsafe.Pointer {
			x := make([]Bar, size)
			return unsafe.Pointer(&x)
		},
		SizeOf:   unsafe.Sizeof([]Bar{}),
		TypeKind: engine.KindSlice,
		Elem:     TypeIdBar,
		TypeId:   TypeIdBarSlice,
	},
	TypeIdBarPtrSlice: {
		Copy: func(dest, from unsafe.Pointer) {
			*(*[]*Bar)(dest) = *(*[]*Bar)(from)
		},
		NewSlice: func(size int) unsafe.Pointer {
			x := make([]*Bar, size)
			return unsafe.Pointer(&x)
		},
		SizeOf:   unsafe.Sizeof([]*Bar{}),
		TypeKind: engine.KindSlice,
		Elem:     TypeIdBarPtr,
		TypeId:   TypeIdBarPtrSlice,
	},
	TypeIdDemoable: {
		Copy: func(dest, from unsafe.Pointer) {
			*(*Demoable)(dest) = *(*Demoable)(from)
		},
		SizeOf:   unsafe.Sizeof(Demoable(nil)),
		TypeKind: engine.KindInterface,
		TypeId:   TypeIdDemoable,
		IntfUnwrap: func(x unsafe.Pointer) (engine.TypeId, unsafe.Pointer) {
			d := *(*Demoable)(x)
			switch t := d.(type) {
			case Bar:
				x := t
				return TypeIdBar, unsafe.Pointer(&x)
			case *Bar:
				x := t
				return TypeIdBarPtr, unsafe.Pointer(&x)
			case *Foo:
				x := t
				return TypeIdFooPtr, unsafe.Pointer(&x)
			case *Demo:
				x := t
				return TypeIdDemoPtr, unsafe.Pointer(&x)
			default:
				return 0, nil
			}
		},
		IntfWrap: func(id engine.TypeId, x unsafe.Pointer) unsafe.Pointer {
			var d Demoable
			switch id {
			case TypeIdBar:
				d = *(*Bar)(x)
			case TypeIdBarPtr:
				d = *(**Bar)(x)
			case TypeIdFooPtr:
				d = *(**Foo)(x)
			case TypeIdDemoPtr:
				d = *(**Demo)(x)
			}
			return unsafe.Pointer(&d)
		},
	},
	TypeIdDemoableSlice: {
		Copy: func(dest, from unsafe.Pointer) {
			*(*[]Demoable)(dest) = *(*[]Demoable)(from)
		},
		NewSlice: func(size int) unsafe.Pointer {
			x := make([]Demoable, size)
			return unsafe.Pointer(&x)
		},
		SizeOf:   unsafe.Sizeof([]Demoable{}),
		TypeKind: engine.KindSlice,
		Elem:     TypeIdDemoable,
		TypeId:   TypeIdDemoableSlice,
	},
}

type WalkerFn func(ctx engine.Context, x Demoable) engine.Decision

func makeData(useIntfs bool) *Demo {
	d := &Demo{
		Foo:         Foo{"foo"},
		FooPtr:      &Foo{"fooPtr"},
		FooSlice:    []Foo{{"fooSlice"}},
		FooPtrSlice: []*Foo{{"fooPtrSlice"}},

		Bar:         Bar{"bar"},
		BarPtr:      &Bar{"barPtr"},
		BarSlice:    []Bar{{"barSlice"}},
		BarPtrSlice: []*Bar{{"barPtrSlice"}},
	}
	d.DemoPtr = d

	if useIntfs {
		d.Demoable = &Bar{"demoable"}
		d.DemoableSlice = []Demoable{
			Bar{"demoableSliceBar"},
			&Bar{"demoableSliceBarPtr"},
			&Foo{"demoableSliceFooPtr"},
		}
	}

	return d
}

func TestBlah(t *testing.T) {
	var w WalkerFn = func(ctx engine.Context, x Demoable) engine.Decision {
		t.Logf("%+v", x)
		switch tx := x.(type) {
		case *Foo:
			tx.Val = "in-place"
		case *Bar:
			return engine.Decision{Replacement: unsafe.Pointer(&Bar{Val: "woot"})}
		}
		return engine.Decision{}
	}

	d := makeData(true)
	e := engine.New(DemoMap)
	x, _, _ := e.Execute(w, TypeIdDemo, unsafe.Pointer(d))
	d2 := *(*Demo)(x)

	t.Logf("%+v", d2)
}

func BenchmarkBlah(b *testing.B) {

	var w WalkerFn = func(ctx engine.Context, x Demoable) engine.Decision {
		return engine.Decision{}
	}
	e := engine.New(DemoMap)

	b.Run("noIntfs", func(b *testing.B) {
		b.ReportAllocs()
		d := makeData(false)
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				_, _, err := e.Execute(w, TypeIdDemo, unsafe.Pointer(d))
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	})
	b.Run("Intfs", func(b *testing.B) {
		b.ReportAllocs()
		d := makeData(true)
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				_, _, err := e.Execute(w, TypeIdDemo, unsafe.Pointer(d))
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	})
}
