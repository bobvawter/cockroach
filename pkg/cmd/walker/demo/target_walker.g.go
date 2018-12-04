//+build !walkerAnalysis

package demo

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"
)

// --------------- Public API ------------------------------------------

// TargetContext allows for in-place structural modification by a
// TargetVisitor.
type TargetContext interface {
	context.Context

	// CanReplace indicates whether or not a call to Replace() can succeed.
	CanReplace() bool
	// Replace will substitute the given value for the value being visited.
	Replace(x Target)

	// CanInsertBefore indicates whether or not a call to InsertBefore() can succeed.
	CanInsertBefore() bool
	// InsertBefore will insert the given value before the value being visited.
	InsertBefore(x Target)

	// CanInsertAfter indicates whether or not a call to InsertAfter() can succeed.
	CanInsertAfter() bool
	// InsertAfter will insert the given value after the value being visited.
	// Note that the inserted value will not be traversed by the visitor.
	InsertAfter(x Target)

	// CanRemove indicates if whether or not a call to Remove() can succeed.
	CanRemove() bool
	// Remove will nullify or delete the value being visited.
	Remove()

	// Stack returns a copy of the objects being visited. This slice
	// is ordered from root to leaf.
	Stack() TargetLocations
	// StackLen returns the current depth of the stack.
	StackLen() int

	// abort will terminate the processing to emit the given error to the user.
	abort(err error)

	// accept processes a value within the context. This method returns a
	// (possibly unchanged) value and whether or not a change occurred
	// in this value or within a nested context. A context should expect
	// accept() to be called more than once on any given instance.
	accept(v TargetVisitor, x interface{}) (result interface{}, changed bool)

	// close will cleanup and recycle the context instance. Contexts
	// which enclose other contexts should propagate the call.
	close()

	// rawStack returns the internal stack structure.
	rawStack() *stackTarget
}

// TargetLocation is reported by TargetContext.Stack().
type TargetLocation struct {
	// Value is the object being visited. This will always be a pointer,
	// even for types that are usually visited using by-value semantics.
	Value Target
	Field string
	Index int
}

// String is for debugging use only.
func (l TargetLocation) String() string {
	var b strings.Builder
	fmt.Fprintf(&b, "%v", l.Value)
	if l.Field != "" {
		fmt.Fprintf(&b, ".%s", l.Field)
		if l.Index >= 0 {
			fmt.Fprintf(&b, "[%d]", l.Index)
		}
	}
	return b.String()
}

type TargetLocations []TargetLocation

// String is for debugging use only.
func (s TargetLocations) String() string {
	var b strings.Builder
	b.WriteString("<root>")
	for _, l := range s {
		if l.Field != "" {
			fmt.Fprintf(&b, ".%s", l.Field)
			if l.Index >= 0 {
				fmt.Fprintf(&b, "[%d]", l.Index)
			}
		}
	}
	return b.String()
}

// This generated interface contains pre/post pairs for
// every struct that implements the Target interface.
type TargetVisitor interface {
	PreByRefType(ctx TargetContext, x *ByRefType) (recurse bool, err error)
	PostByRefType(ctx TargetContext, x *ByRefType) error
	PreByValType(ctx TargetContext, x ByValType) (recurse bool, err error)
	PostByValType(ctx TargetContext, x ByValType) error
	PreContainerType(ctx TargetContext, x *ContainerType) (recurse bool, err error)
	PostContainerType(ctx TargetContext, x *ContainerType) error
}

// A default implementation of TargetVisitor.
// This has provisions for allowing users to provide default
// pre/post methods.
type TargetVisitorBase struct {
	DefaultPre  func(ctx TargetContext, x Target) (recurse bool, err error)
	DefaultPost func(ctx TargetContext, x Target) error
}

var _ TargetVisitor = &TargetVisitorBase{}

// PreByRefType implements the TargetVisitor interface.
func (b TargetVisitorBase) PreByRefType(ctx TargetContext, x *ByRefType) (recurse bool, err error) {
	if b.DefaultPre == nil {
		return true, nil
	}
	return b.DefaultPre(ctx, x)
}

// PostByRefType implements the TargetVisitor interface.
func (b TargetVisitorBase) PostByRefType(ctx TargetContext, x *ByRefType) error {
	if b.DefaultPost == nil {
		return nil
	}
	return b.DefaultPost(ctx, x)
}

// PreByValType implements the TargetVisitor interface.
func (b TargetVisitorBase) PreByValType(ctx TargetContext, x ByValType) (recurse bool, err error) {
	if b.DefaultPre == nil {
		return true, nil
	}
	return b.DefaultPre(ctx, x)
}

// PostByValType implements the TargetVisitor interface.
func (b TargetVisitorBase) PostByValType(ctx TargetContext, x ByValType) error {
	if b.DefaultPost == nil {
		return nil
	}
	return b.DefaultPost(ctx, x)
}

// PreContainerType implements the TargetVisitor interface.
func (b TargetVisitorBase) PreContainerType(
	ctx TargetContext, x *ContainerType,
) (recurse bool, err error) {
	if b.DefaultPre == nil {
		return true, nil
	}
	return b.DefaultPre(ctx, x)
}

// PostContainerType implements the TargetVisitor interface.
func (b TargetVisitorBase) PostContainerType(ctx TargetContext, x *ContainerType) error {
	if b.DefaultPost == nil {
		return nil
	}
	return b.DefaultPost(ctx, x)
}

// --------------------------- Base Context ----------------------------

// A baseTargetContext implements an immutable TargetContext.
// A single instance will be shared across all derived contexts
// to transmit common state.
type baseTargetContext struct {
	context.Context
	stack *stackTarget
}

var _ TargetContext = &baseTargetContext{}

func (c *baseTargetContext) accept(v TargetVisitor, x interface{}) (interface{}, bool) {
	return x, false
}

func (c *baseTargetContext) abort(err error) {
	err = &TargetWalkError{reason: err, stack: c.Stack()}
	panic(err)
}

func (*baseTargetContext) close() {}

func (*baseTargetContext) CanReplace() bool {
	return false
}

func (c *baseTargetContext) Replace(n Target) {
	c.abort(errors.New("this context cannot replace"))
}

func (c *baseTargetContext) replace(n traversableTarget) {
	c.abort(errors.New("this context cannot replace"))
}

func (*baseTargetContext) CanInsertBefore() bool {
	return false
}

func (c *baseTargetContext) InsertBefore(n Target) {
	c.abort(errors.New("this context cannot insert"))
}

func (*baseTargetContext) CanInsertAfter() bool {
	return false
}

func (c *baseTargetContext) InsertAfter(n Target) {
	c.abort(errors.New("this context cannot insert"))
}

func (*baseTargetContext) CanRemove() bool {
	return false
}

func (c *baseTargetContext) rawStack() *stackTarget {
	return c.stack
}

func (c *baseTargetContext) Remove() {
	c.abort(errors.New("this context cannot remove"))
}

func (c *baseTargetContext) Stack() TargetLocations {
	return c.stack.Copy()
}

func (c *baseTargetContext) StackLen() int {
	return c.stack.Len()
}

// ------------------------- Type enhancements -------------------------

// traversableTarget are the methods that we will add to traversable types.
type traversableTarget interface {
	// pre calls the relevant PreXYZ method on the visitor.
	pre(ctx TargetContext, v TargetVisitor) (bool, error)
	// post calls the relevant PreXYZ method on the visitor.
	post(ctx TargetContext, v TargetVisitor) error
	// traverse visits the fields within the struct.
	traverse(ctx TargetContext, v TargetVisitor)
}

// Ensure we implement the traversableTarget interface.
var (
	_ traversableTarget = &ByRefType{}
	_ traversableTarget = ByValType{}
	_ traversableTarget = &ContainerType{}
)

func (x *ByRefType) pre(ctx TargetContext, v TargetVisitor) (bool, error) {
	return v.PreByRefType(ctx, x)
}
func (x *ByRefType) post(ctx TargetContext, v TargetVisitor) error {
	return v.PostByRefType(ctx, x)
}
func (x *ByRefType) traverse(ctx TargetContext, v TargetVisitor) {}
func (x ByValType) pre(ctx TargetContext, v TargetVisitor) (bool, error) {
	return v.PreByValType(ctx, x)
}
func (x ByValType) post(ctx TargetContext, v TargetVisitor) error {
	return v.PostByValType(ctx, x)
}
func (x ByValType) traverse(ctx TargetContext, v TargetVisitor) {}
func (x *ContainerType) pre(ctx TargetContext, v TargetVisitor) (bool, error) {
	return v.PreContainerType(ctx, x)
}
func (x *ContainerType) post(ctx TargetContext, v TargetVisitor) error {
	return v.PostContainerType(ctx, x)
}
func (x *ContainerType) traverse(ctx TargetContext, v TargetVisitor) {
	top := ctx.rawStack().Top()
	dirty := false

	// This code is structured to avoid escaping of the
	// newLocalVariables by not passing them into accept(), but
	// by using an x.FieldReference, instead.

	newByRef /* *ByRefType */ := &x.ByRef
	{
		top.Field = "ByRef"
		c := buildTargetContext(ctx, newScalarTargetContext(), newTypeCheckTargetContext(typeOfByRefType))
		if y, changed := c.accept(v, &x.ByRef); changed {
			dirty = true
			newByRef = y.(*ByRefType)
		}
		c.close()
	}
	newByRefPtr /* *ByRefType */ := x.ByRefPtr
	if newByRefPtr != nil {
		top.Field = "ByRefPtr"
		c := buildTargetContext(ctx, newScalarTargetContext(), newTypeCheckTargetContext(typeOfByRefTypePtr))
		if y, changed := c.accept(v, x.ByRefPtr); changed {
			dirty = true
			newByRefPtr = y.(*ByRefType)
		}
		c.close()
	}
	newByRefSlice /* []ByRefType */ := x.ByRefSlice
	if newByRefSlice != nil {
		top.Field = "ByRefSlice"
		c := buildTargetContext(ctx, newSliceTargetContext(), newScalarTargetContext(), newTypeCheckTargetContext(typeOfByRefType))
		if y, changed := c.accept(v, x.ByRefSlice); changed {
			dirty = true
			newByRefSlice = y.([]ByRefType)
		}
		c.close()
	}
	newByRefPtrSlice /* []*ByRefType */ := x.ByRefPtrSlice
	if newByRefPtrSlice != nil {
		top.Field = "ByRefPtrSlice"
		c := buildTargetContext(ctx, newSliceTargetContext(), newScalarTargetContext(), newTypeCheckTargetContext(typeOfByRefTypePtr))
		if y, changed := c.accept(v, x.ByRefPtrSlice); changed {
			dirty = true
			newByRefPtrSlice = y.([]*ByRefType)
		}
		c.close()
	}
	newByVal /* ByValType */ := x.ByVal
	{
		top.Field = "ByVal"
		c := buildTargetContext(ctx, newScalarTargetContext(), newTypeCheckTargetContext(typeOfByValType))
		if y, changed := c.accept(v, x.ByVal); changed {
			dirty = true
			newByVal = y.(ByValType)
		}
		c.close()
	}
	newByValPtr /* *ByValType */ := x.ByValPtr
	if newByValPtr != nil {
		top.Field = "ByValPtr"
		c := buildTargetContext(ctx, newPointerTargetContext(false), newScalarTargetContext(), newTypeCheckTargetContext(typeOfByValType))
		if y, changed := c.accept(v, x.ByValPtr); changed {
			dirty = true
			newByValPtr = y.(*ByValType)
		}
		c.close()
	}
	newByValSlice /* []*ByValType */ := x.ByValSlice
	if newByValSlice != nil {
		top.Field = "ByValSlice"
		c := buildTargetContext(ctx, newSliceTargetContext(), newPointerTargetContext(false), newScalarTargetContext(), newTypeCheckTargetContext(typeOfByValType))
		if y, changed := c.accept(v, x.ByValSlice); changed {
			dirty = true
			newByValSlice = y.([]*ByValType)
		}
		c.close()
	}
	newByValPtrSlice /* []*ByValType */ := x.ByValPtrSlice
	if newByValPtrSlice != nil {
		top.Field = "ByValPtrSlice"
		c := buildTargetContext(ctx, newSliceTargetContext(), newPointerTargetContext(false), newScalarTargetContext(), newTypeCheckTargetContext(typeOfByValType))
		if y, changed := c.accept(v, x.ByValPtrSlice); changed {
			dirty = true
			newByValPtrSlice = y.([]*ByValType)
		}
		c.close()
	}
	newContainer /* *ContainerType */ := x.Container
	if newContainer != nil {
		top.Field = "Container"
		c := buildTargetContext(ctx, newScalarTargetContext(), newTypeCheckTargetContext(typeOfContainerTypePtr))
		if y, changed := c.accept(v, x.Container); changed {
			dirty = true
			newContainer = y.(*ContainerType)
		}
		c.close()
	}
	newEmbedsTarget /* EmbedsTarget */ := x.EmbedsTarget
	if newEmbedsTarget != nil {
		top.Field = "EmbedsTarget"
		c := buildTargetContext(ctx, newScalarTargetContext(), newTypeCheckTargetContext(typeOfEmbedsTarget))
		if y, changed := c.accept(v, x.EmbedsTarget); changed {
			dirty = true
			newEmbedsTarget = y.(EmbedsTarget)
		}
		c.close()
	}
	newEmbedsTargetPtr /* *EmbedsTarget */ := x.EmbedsTargetPtr
	if newEmbedsTargetPtr != nil {
		top.Field = "EmbedsTargetPtr"
		c := buildTargetContext(ctx, newPointerTargetContext(false), newScalarTargetContext(), newTypeCheckTargetContext(typeOfEmbedsTarget))
		if y, changed := c.accept(v, x.EmbedsTargetPtr); changed {
			dirty = true
			newEmbedsTargetPtr = y.(*EmbedsTarget)
		}
		c.close()
	}
	newAnonymousTargetSlice /* []Target */ := x.AnonymousTargetSlice
	if newAnonymousTargetSlice != nil {
		top.Field = "AnonymousTargetSlice"
		c := buildTargetContext(ctx, newSliceTargetContext(), newScalarTargetContext(), newTypeCheckTargetContext(typeOfTarget))
		if y, changed := c.accept(v, x.AnonymousTargetSlice); changed {
			dirty = true
			newAnonymousTargetSlice = y.([]Target)
		}
		c.close()
	}
	newNamedTargets /* Targets */ := x.NamedTargets
	if newNamedTargets != nil {
		top.Field = "NamedTargets"
		c := buildTargetContext(ctx, newSliceTargetContext(), newScalarTargetContext(), newTypeCheckTargetContext(typeOfTarget))
		if y, changed := c.accept(v, x.NamedTargets); changed {
			dirty = true
			newNamedTargets = y.(Targets)
		}
		c.close()
	}
	newInterfacePtrSlice /* []*Target */ := x.InterfacePtrSlice
	if newInterfacePtrSlice != nil {
		top.Field = "InterfacePtrSlice"
		c := buildTargetContext(ctx, newSliceTargetContext(), newPointerTargetContext(false), newScalarTargetContext(), newTypeCheckTargetContext(typeOfTarget))
		if y, changed := c.accept(v, x.InterfacePtrSlice); changed {
			dirty = true
			newInterfacePtrSlice = y.([]*Target)
		}
		c.close()
	}

	// Clear field for Post().
	top.Field = ""

	if dirty && ctx.CanReplace() {
		ctx.Replace(&ContainerType{
			ByRef:                *newByRef,
			ByRefPtr:             newByRefPtr,
			ByRefSlice:           newByRefSlice,
			ByRefPtrSlice:        newByRefPtrSlice,
			ByVal:                newByVal,
			ByValPtr:             newByValPtr,
			ByValSlice:           newByValSlice,
			ByValPtrSlice:        newByValPtrSlice,
			Container:            newContainer,
			EmbedsTarget:         newEmbedsTarget,
			EmbedsTargetPtr:      newEmbedsTargetPtr,
			AnonymousTargetSlice: newAnonymousTargetSlice,
			NamedTargets:         newNamedTargets,
			InterfacePtrSlice:    newInterfacePtrSlice,
		})
	}
}

// --------------- Error types -----------------------------------------

// TargetWalkError is used to communicate errors that occur during a
// traversal. This type provides access to a snapshot of the stack
// when the error occurred to aid in debugging.
type TargetWalkError struct {
	reason error
	stack  TargetLocations
}

var _ error = &TargetWalkError{}

// Cause returns the causal error.
func (e *TargetWalkError) Cause() error {
	return e.reason
}

// Error implements the error interface.
func (e *TargetWalkError) Error() string {
	return e.reason.Error()
}

// Stack returns a snapshot of the visitation stack where the enclosed
// error occurred.
func (e *TargetWalkError) Stack() TargetLocations {
	return e.stack
}

// String is for debugging use only.
func (e *TargetWalkError) String() string {
	return fmt.Sprintf("%v at %v", e.reason, e.stack)
}

// ------------------- Context Factory ---------------------------------

// buildTargetContext sets up the linkages between contexts.
// The "current" context must be the zeroth element, meaning that
// stack must have at least two elements.
func buildTargetContext(stack ...TargetContext) TargetContext {
	// Fold the entries together, linking them to their parent context.
	for i, l := 1, len(stack); i < l; i++ {
		switch t := stack[i].(type) {
		case *pointerTargetContext:
			t.TargetContext = stack[i-1]
			t.elementContext = stack[i+1]
		case *sliceTargetContext:
			t.TargetContext = stack[i-1]
			t.elementContext = stack[i+1]
		case *scalarTargetContext:
			t.TargetContext = stack[i-1]
			t.elementContext = stack[i+1]
		case *typeCheckTargetContext:
			t.TargetContext = stack[i-1]
		default:
			panic(fmt.Errorf("unsupported type: %+v", t))
		}
	}
	return stack[1]
}

var scalarTargetContextPool = sync.Pool{
	New: func() interface{} { return &scalarTargetContext{} },
}

func newScalarTargetContext() TargetContext {
	ret := scalarTargetContextPool.Get().(*scalarTargetContext)
	*ret = scalarTargetContext{}
	return ret
}

var pointerTargetContextPool = sync.Pool{
	New: func() interface{} { return &pointerTargetContext{} },
}

func newPointerTargetContext(takeAddr bool) TargetContext {
	ret := pointerTargetContextPool.Get().(*pointerTargetContext)
	*ret = pointerTargetContext{takeAddr: takeAddr}
	return ret
}

var sliceTargetContextPool = sync.Pool{
	New: func() interface{} { return &sliceTargetContext{} },
}

func newSliceTargetContext() TargetContext {
	ret := sliceTargetContextPool.Get().(*sliceTargetContext)
	*ret = sliceTargetContext{}
	return ret
}

var typeCheckTargetContextPool = sync.Pool{
	New: func() interface{} { return &typeCheckTargetContext{} },
}

func newTypeCheckTargetContext(assignableTo reflect.Type) TargetContext {
	ret := typeCheckTargetContextPool.Get().(*typeCheckTargetContext)
	*ret = typeCheckTargetContext{assignableTo: assignableTo}
	return ret
}

// ----------------------- Pointer Context -----------------------------

// pointerTargetContext is used for impedence matching when we
// have a by-value type in a by-ref field.
type pointerTargetContext struct {
	TargetContext
	elementContext TargetContext
	takeAddr       bool
}

var _ TargetContext = &pointerTargetContext{}

func (c *pointerTargetContext) accept(v TargetVisitor, val interface{}) (interface{}, bool) {
	if c.takeAddr {
		x := reflect.ValueOf(val).Addr()
		// Handle "typed nils" like SomeIntf(nil).
		if x.IsValid() {
			intf := x.Interface()
			if intf != nil {
				if y, changed := c.elementContext.accept(v, x.Interface()); changed {
					return y.Elem().Interface(), true
				}
			}
		}
	} else {
		x := reflect.ValueOf(val).Elem()
		// Handle "typed nils" like SomeIntf(nil).
		if x.IsValid() {
			intf := x.Interface()
			if intf != nil {
				if y, changed := c.elementContext.accept(v, x.Interface()); changed {
					x.Set(reflect.ValueOf(y))
					return val, true
				}
			}
		}
	}
	return val, false
}

func (c *pointerTargetContext) close() {
	c.elementContext.close()
	*c = pointerTargetContext{}
	pointerTargetContextPool.Put(c)
}

// ------------------------ Scalar Context -----------------------------

// scalarTargetContext instances should be obtained through
// TargetContext.scalarContext().
type scalarTargetContext struct {
	TargetContext
	dirty          bool
	elementContext TargetContext
	replacement    interface{}
}

var _ TargetContext = &scalarTargetContext{}

func (c *scalarTargetContext) accept(
	v TargetVisitor, val interface{},
) (result interface{}, changed bool) {
	c.dirty = false

	if x, changed := c.elementContext.accept(v, val); changed {
		return x, true
	}
	if c.dirty {
		return c.replacement, true
	}
	return val, false
}

func (c *scalarTargetContext) close() {
	c.elementContext.close()
	*c = scalarTargetContext{}
	scalarTargetContextPool.Put(c)
}

func (*scalarTargetContext) CanReplace() bool {
	return true
}
func (c *scalarTargetContext) Replace(n Target) {
	c.dirty = true
	c.replacement = n.(traversableTarget)
}

// ------------------------ Slice Context ------------------------------

// sliceTargetContext instances should be obtained through
// TargetContext.sliceContext().
type sliceTargetContext struct {
	TargetContext
	didRemove      bool
	didReplace     bool
	dirty          bool
	elementContext TargetContext
	insertAfter    []interface{}
	insertBefore   []interface{}
	replacement    interface{}
}

func (c *sliceTargetContext) accept(
	v TargetVisitor, val interface{},
) (result interface{}, changed bool) {
	dirty := false
	// We defer initialization until a modification is made.
	var out reflect.Value
	slice := reflect.ValueOf(val)
	top := c.rawStack().Top()

	for i, l := 0, slice.Len(); i < l; i++ {
		elt := slice.Index(i).Interface()

		if elt == nil {
			// Preserve nil elements.
			if dirty {
				out = reflect.Append(out, slice.Index(i))
			}
			continue
		}

		top.Index = i
		elt, changed := c.elementContext.accept(v, elt)
		if changed {
			c.dirty = true
			c.didReplace = true
			c.replacement = elt
		}

		if !dirty {
			if c.dirty {
				dirty = true
				// Create and backfill our result slice.
				out = reflect.MakeSlice(slice.Type(), 0, l)
				out = reflect.AppendSlice(out, slice.Slice(0, i))
			} else {
				continue
			}
		}

		if c.insertBefore != nil {
			for _, i := range c.insertBefore {
				out = reflect.Append(out, reflect.ValueOf(i))
			}
			c.insertBefore = nil
		}
		// We check for elt == nil above, so if we're seeing a nil here,
		// it means that the user removed the element.
		if c.didRemove {
			c.didRemove = false
		} else if c.didReplace {
			c.didReplace = false
			out = reflect.Append(out, reflect.ValueOf(c.replacement))
			c.replacement = nil
		} else {
			out = reflect.Append(out, reflect.ValueOf(elt))
		}
		if c.insertAfter != nil {
			for _, i := range c.insertAfter {
				out = reflect.Append(out, reflect.ValueOf(i))
			}
			c.insertAfter = nil
		}
	}

	top.Index = -1

	if dirty {
		val = out.Interface()
	}
	return val, c.dirty
}

func (c *sliceTargetContext) close() {
	c.elementContext.close()
	// Nullify any references and push back into pool.
	*c = sliceTargetContext{}
	sliceTargetContextPool.Put(c)
}

func (c *sliceTargetContext) CanInsertAfter() bool {
	return true
}
func (c *sliceTargetContext) InsertAfter(val Target) {
	c.dirty = true
	c.insertAfter = append(c.insertAfter, val)
}
func (c *sliceTargetContext) CanInsertBefore() bool {
	return true
}
func (c *sliceTargetContext) InsertBefore(val Target) {
	c.dirty = true
	c.insertBefore = append(c.insertBefore, val)
}
func (c *sliceTargetContext) CanRemove() bool {
	return true
}
func (c *sliceTargetContext) Remove() {
	c.dirty = true
	c.didRemove = true
	c.didReplace = false
}
func (c *sliceTargetContext) CanReplace() bool {
	return true
}
func (c *sliceTargetContext) Replace(x Target) {
	c.dirty = true
	c.didRemove = false
	c.didReplace = true
	c.replacement = x
}

// ------------------------ Stack Support ------------------------------

// stackTarget is a datastructure that's intended to be shared across
// derived instances of TargetContext to allow the backing array
// to be reused.
type stackTarget TargetLocations

func newstackTarget() *stackTarget {
	s := make(stackTarget, 0, 32)
	return &s
}

// Copy duplicates the stack.
func (s *stackTarget) Copy() TargetLocations {
	return TargetLocations(append((*s)[:0:0], *s...))
}

// Push adds a new location to the stack.
func (s *stackTarget) Push(loc TargetLocation) {
	*s = append(*s, loc)
}

// Pop removes the top element from the stack.  It will panic if
// an empty stack is popped.
func (s *stackTarget) Pop() {
	*s = (*s)[:len(*s)-1]
}

// Len returns the length of the stack.
func (s *stackTarget) Len() int {
	return len(*s)
}

// Reset will zero out the stack, retaining the backing array if
// it hasn't grown beyond the default size.
func (s *stackTarget) Reset() *stackTarget {
	*s = (*s)[:0:32]
	return s
}

// Top returns a pointer to the top frame of the stack so that it may
// be modified in-place.
func (s *stackTarget) Top() *TargetLocation {
	return &((*s)[len(*s)-1])
}

// ------------------- Support code ------------------------------------

// reflect.Type symbols for helping with type-safety.
var (
	typeOfByRefTypePtr      = reflect.TypeOf([]*ByRefType(nil)).Elem()
	typeOfByValTypePtr      = reflect.TypeOf([]*ByValType(nil)).Elem()
	typeOfContainerTypePtr  = reflect.TypeOf([]*ContainerType(nil)).Elem()
	typeOfEmbedsTargetPtr   = reflect.TypeOf([]*EmbedsTarget(nil)).Elem()
	typeOfTargetPtr         = reflect.TypeOf([]*Target(nil)).Elem()
	typeOfByRefType         = reflect.TypeOf([]*ByRefType(nil)).Elem()
	typeOfByValType         = reflect.TypeOf([]ByValType(nil)).Elem()
	typeOfContainerType     = reflect.TypeOf([]*ContainerType(nil)).Elem()
	typeOfEmbedsTarget      = reflect.TypeOf([]EmbedsTarget(nil)).Elem()
	typeOfTarget            = reflect.TypeOf([]Target(nil)).Elem()
	typeOfTargets           = reflect.TypeOf([]Targets(nil)).Elem()
	typeOfByRefTypePtrSlice = reflect.TypeOf([][]*ByRefType(nil)).Elem()
	typeOfByValTypePtrSlice = reflect.TypeOf([][]*ByValType(nil)).Elem()
	typeOfTargetPtrSlice    = reflect.TypeOf([][]*Target(nil)).Elem()
	typeOfByRefTypeSlice    = reflect.TypeOf([][]ByRefType(nil)).Elem()
	typeOfTargetSlice       = reflect.TypeOf([][]Target(nil)).Elem()
)

// ------------------- Type Checking Context ---------------------------

// typeCheckTargetContext is the context facade exposed to user-code.
// It is also responsible for pushing and popping TargetLocation.
type typeCheckTargetContext struct {
	TargetContext
	assignableTo reflect.Type
}

var _ TargetContext = &typeCheckTargetContext{}

func (c *typeCheckTargetContext) accept(
	v TargetVisitor, val interface{},
) (result interface{}, changed bool) {
	c.rawStack().Push(TargetLocation{
		Value: val.(Target),
		Index: -1,
	})

	x := val.(traversableTarget)
	recurse, err := x.pre(c, v)
	if err != nil {
		c.abort(err)
	}
	if recurse {
		x.traverse(c, v)
	}
	if err := x.post(c, v); err != nil {
		c.abort(err)
	}
	c.rawStack().Pop()
	return x, false
}

// check verifies that val is assignable to the configured type.
func (c *typeCheckTargetContext) check(val Target) {
	valTyp := reflect.TypeOf(val)
	if !valTyp.AssignableTo(c.assignableTo) {
		c.abort(fmt.Errorf("%s is not assignable to %s", valTyp, c.assignableTo))
	}
}

func (c *typeCheckTargetContext) close() {
	*c = typeCheckTargetContext{}
	typeCheckTargetContextPool.Put(c)
}

func (c *typeCheckTargetContext) InsertAfter(n Target) {
	c.check(n)
	c.TargetContext.InsertAfter(n)
}

func (c *typeCheckTargetContext) InsertBefore(n Target) {
	c.check(n)
	c.TargetContext.InsertBefore(n)
}

func (c *typeCheckTargetContext) Replace(n Target) {
	c.check(n)
	c.TargetContext.Replace(n)
}

// ------------- Walk functions ----------------------------------------

// Pool the root contexts, which own the stack slices.
var rootTargetContextPool = sync.Pool{New: func() interface{} {
	return &baseTargetContext{stack: newstackTarget()}
}}

// walk provides a top-level behavior for setting up a root context
// and for unwinding the stack after a panic.
func walkTarget(
	ctx context.Context, v TargetVisitor, tgt Target, assignableTo reflect.Type,
) (res traversableTarget, changed bool, err error) {
	root := rootTargetContextPool.Get().(*baseTargetContext)
	root.Context = ctx

	defer func() {
		// Reset and trim capacity.
		*root = baseTargetContext{stack: root.stack.Reset()}
		rootTargetContextPool.Put(root)
		if r := recover(); r != nil {
			if we, ok := r.(*TargetWalkError); ok {
				err = we
			} else {
				panic(r)
			}
		}
	}()

	c := buildTargetContext(
		root,
		newScalarTargetContext(),
		newTypeCheckTargetContext(assignableTo),
	)
	x, changed := c.accept(v, tgt)
	c.close()
	res = x.(traversableTarget)
	return
}

// WalkEmbedsTarget walks a visitor over an EmbedsTarget.
func WalkEmbedsTarget(
	ctx context.Context, v TargetVisitor, tgt EmbedsTarget,
) (result EmbedsTarget, changed bool, err error) {
	t, changed, err := walkTarget(ctx, v, tgt, typeOfEmbedsTarget)
	result = t.(EmbedsTarget)
	return
}

// WalkTarget walks a visitor over an Target.
func WalkTarget(
	ctx context.Context, v TargetVisitor, tgt Target,
) (result Target, changed bool, err error) {
	t, changed, err := walkTarget(ctx, v, tgt, typeOfTarget)
	result = t.(Target)
	return
}

// Walk applies the visitor to the ByRefType. It returns the original
// value if none of the TargetContext mutation methods were changed,
// or a replacement value if they were.
func (x *ByRefType) Walk(
	ctx context.Context, v TargetVisitor,
) (result *ByRefType, changed bool, err error) {
	impl, changed, err := walkTarget(ctx, v, x, typeOfByRefType)
	if err != nil {
		return
	}
	result = impl.(*ByRefType)
	return
}

// Walk applies the visitor to the ByValType. It returns the original
// value if none of the TargetContext mutation methods were changed,
// or a replacement value if they were.
func (x ByValType) Walk(
	ctx context.Context, v TargetVisitor,
) (result ByValType, changed bool, err error) {
	impl, changed, err := walkTarget(ctx, v, x, typeOfByValType)
	if err != nil {
		return
	}
	result = impl.(ByValType)
	return
}

// Walk applies the visitor to the ContainerType. It returns the original
// value if none of the TargetContext mutation methods were changed,
// or a replacement value if they were.
func (x *ContainerType) Walk(
	ctx context.Context, v TargetVisitor,
) (result *ContainerType, changed bool, err error) {
	impl, changed, err := walkTarget(ctx, v, x, typeOfContainerType)
	if err != nil {
		return
	}
	result = impl.(*ContainerType)
	return
}
