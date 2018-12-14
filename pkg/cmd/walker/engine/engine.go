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

// Package engine holds base implementation details for use by
// generated code. Users should not depend on any particular feature
// of this package.
package engine

import (
	"fmt"
	"reflect"
	"unsafe"
)

// See discussion on frame.Slots.
const fixedSlotCount = 16

// A TypeId is an opaque reference to a visitable type.
type TypeId int

// A TypeMap holds the necessary metadata to visit a collection of types.
type TypeMap []*TypeData

// A Kind determines the dispatch strategy for a given visitable type.
type Kind int

// FacadeFn is a generated function type that depends on the visitable
// interface.
type FacadeFn interface{}

// A visitable type has some combinations of kinds which determine
// its access pattern.
const (
	_ Kind = iota
	KindInterface
	KindPointer
	KindSlice
	KindStruct
)

type TypeData struct {
	Copy       func(dest, from unsafe.Pointer)
	Elem       TypeId
	Facade     func(Context, FacadeFn, unsafe.Pointer) Decision
	Fields     []FieldInfo
	IntfUnwrap func(unsafe.Pointer) (TypeId, unsafe.Pointer)
	IntfWrap   func(TypeId, unsafe.Pointer) unsafe.Pointer
	New        func() unsafe.Pointer
	NewSlice   func(size int) unsafe.Pointer
	SizeOf     uintptr
	TypeKind   Kind
	TypeId     TypeId
}

type FieldInfo struct {
	Name   string
	Offset uintptr
	Target TypeId
}

type Context struct{}

type Decision struct {
	Replacement unsafe.Pointer
	Post        FacadeFn
}

// A stack is a collection of frames.
type stack []frame

// A frame represents the visitation of a single struct,
// interface, or slice.
type frame struct {
	// Count holds the number of slots to be visited.
	Count int
	// Idx is the current slot being visited.
	Idx int
	// We keep a fixed-size array of slots per level so that most
	// visitable objects won't need a heap allocation to store
	// the intermediate state.
	Slots [fixedSlotCount]slot
	// Large targets (such as slices) will use additional, heap-allocated
	// memory to store the intermediate state.
	Overflow []slot
}

// newFrame constructs a frame with at least count Slots.
func newFrame(count int) frame {
	l := frame{Count: count}
	if count > fixedSlotCount {
		l.Overflow = make([]slot, count-fixedSlotCount)
	}
	return l
}

// Slot is used to access a storage slot within the frame.
func (f *frame) Slot(idx int) *slot {
	if idx < fixedSlotCount {
		return &f.Slots[idx]
	} else {
		return &f.Overflow[idx-fixedSlotCount]
	}
}

// SetSlot is a helper function to configure a slot.
func (f *frame) SetSlot(idx int, td *TypeData, x unsafe.Pointer) {
	*f.Slot(idx) = slot{TypeData: td, Value: x}
}

// A slot represents storage space for visitable object, such as a field
// within a struct, or an element of a slice.
type slot struct {
	Dirty    bool
	TypeData *TypeData
	Value    unsafe.Pointer
}

// An Engine holds the necessary information to pass a visitor over
// a field.
type Engine struct {
	typeMap TypeMap
}

func New(m TypeMap) *Engine {
	return &Engine{typeMap: m}
}

// Execute drives the visitation process. This function is essentially
// an unrolled loop that maintains its own stack to avoid deeply-nested
// call stacks. We can also perform cycle-detection.
func (e *Engine) Execute(fn FacadeFn, t TypeId, x unsafe.Pointer) (unsafe.Pointer, bool, error) {
	stack := make(stack, 2, 8)

	// We ignore stack[0]; it exists only so that we don't need to
	// special-case the top-frame frame not having a parent.
	stack[1].Count = 1
	stack[1].SetSlot(0, e.typeMap[t], x)
	var returning *frame

top:
	// Determine where we are and what the current type of the slot is.
	stackIdx := len(stack) - 1
	frame := &stack[stackIdx]
	s := frame.Slot(frame.Idx)
	td := s.TypeData

	// Once we've processed every slot in a frame, we start an unwinding
	// process that propagates changes upwards in the visitation stack.
	if returning != nil {
		goto unwind
	}

	// Linear search for cycle-breaking. Note that this does not guarantee
	// exactly-once behavior. pprof says this is much faster than using a
	// map structure, especially since we expect the stack to be of
	// reasonable depth. We use the type and pointer as a unique key. We
	// need to be able to distinguish a struct from the first field of the
	// struct. Remember, too, that go disallows recursive type
	// definitions.
	for l := 1; l < stackIdx; l++ {
		onStack := stack[l].Slot(stack[l].Idx)
		if onStack.Value == s.Value && onStack.TypeData == td {
			goto skipSlot
		}
	}

	// In this switch statement, we're going to drill into a slot,
	// which might cause another frame to be pushed onto the stack
	// if it's a composite value.
	switch td.TypeKind {
	case KindPointer:
		// We dereference the pointer and push the resulting memory
		// location as a 1-slot frame.
		ptr := *(*unsafe.Pointer)(s.Value)
		if ptr == nil {
			goto unwind
		}
		next := newFrame(1)
		next.SetSlot(0, e.typeMap[td.Elem], ptr)
		stack = append(stack, next)

	case KindStruct:
		// Structs are where we call out to user logic via a generated,
		// type-safe facade. The user code can trigger various flow-control
		// to happen.
		d := td.Facade(Context{}, fn, s.Value)
		if d.Replacement != nil {
			s.Dirty = true
			s.Value = d.Replacement
		}
		// Slices and structs have very similar approaches, we create a new
		// frame and create slots for each field or slice element.
		fieldCount := len(td.Fields)
		if fieldCount == 0 {
			goto unwind
		}
		next := newFrame(fieldCount)
		for i, f := range td.Fields {
			fPtr := unsafe.Pointer(uintptr(s.Value) + f.Offset)
			next.SetSlot(i, e.typeMap[f.Target], fPtr)
		}
		stack = append(stack, next)

	case KindSlice:
		// Slices have the same general flow as a struct; they're just
		// a sequence of visitable values.
		header := *(*reflect.SliceHeader)(s.Value)
		if header.Len == 0 {
			goto unwind
		}
		next := newFrame(header.Len)
		eltTd := e.typeMap[td.Elem]
		for i := 0; i < header.Len; i++ {
			next.SetSlot(i, eltTd, unsafe.Pointer(header.Data+uintptr(i)*eltTd.SizeOf))
		}
		stack = append(stack, next)

	case KindInterface:
		elem, ptr := td.IntfUnwrap(s.Value)
		if ptr == nil {
			goto unwind
		}
		next := newFrame(1)
		next.SetSlot(0, e.typeMap[elem], ptr)
		stack = append(stack, next)

	default:
		panic(fmt.Sprintf("unimplemented: %d", td.TypeKind))
	}

	// We've pushed a new frame onto the stack, so we'll restart.
	goto top

unwind:
	// If the slot reports that it's dirty, we want to propagate
	// the changes upwards in the stack.
	if s.Dirty {
		// Propagate dirtiness.
		parentLevel := &stack[stackIdx-1]
		parentFrame := parentLevel.Slot(parentLevel.Idx)
		parentFrame.Dirty = true

		// This switch statement is the inverse of the above.
		switch td.TypeKind {
		case KindStruct:
			// Allocate a replacement instance of the struct.
			next := td.New()
			// Perform a shallow copy to catch non-visitable fields.
			td.Copy(next, s.Value)

			// Copy the visitable fields into the new struct.
			for i, f := range td.Fields {
				fPtr := unsafe.Pointer(uintptr(next) + f.Offset)
				e.typeMap[f.Target].Copy(fPtr, returning.Slot(i).Value)
			}
			s.Value = next

		case KindPointer:
			// Copy out the pointer to a local var so we don't stomp on it.
			next := returning.Slot(0).Value
			s.Value = unsafe.Pointer(&next)

		case KindSlice:
			// Create a new slice instance and populate the elements.
			next := td.NewSlice(returning.Count)
			toHeader := *(*reflect.SliceHeader)(next)
			elemTd := e.typeMap[td.Elem]

			// Copy the elements across.
			for i := 0; i < returning.Count; i++ {
				toElem := unsafe.Pointer(toHeader.Data + uintptr(i)*elemTd.SizeOf)
				elemTd.Copy(toElem, returning.Slot(i).Value)
			}
			s.Value = next

		case KindInterface:
			next := returning.Slot(0)
			s.Value = td.IntfWrap(next.TypeData.TypeId, next.Value)

		default:
			panic(fmt.Sprintf("unimplemented: %d", td.TypeKind))
		}
	}

skipSlot:
	frame.Idx++
	if frame.Idx == frame.Count {
		if stackIdx == 1 {
			return frame.Slot(0).Value, frame.Slot(0).Dirty, nil
		}
		returning = frame
		stack = stack[:stackIdx]
	} else {
		returning = nil
	}
	goto top
}
