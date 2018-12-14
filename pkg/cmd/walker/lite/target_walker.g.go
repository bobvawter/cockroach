//+build !walkerAnalysis

package lite

import (
	"errors"
	"fmt"
)

type TargetImpl interface {
	// ChildAt returns the N-th child of an TargetImpl. For struct kinds,
	// this will be the N-th visitable field of the struct. For slices,
	// this will be the N-th element in the slice.
	ChildAt(index int) TargetImpl
	// ChildNamed provides by-name access to fields. This allows types
	// to implement by-convention protocols.
	ChildNamed(name string) (_ TargetImpl, ok bool)
	// TargetKind returns a type token to identify the TargetImpl.
	TargetKind() TargetKind
	// NumChildren returns the number of children in the TargetImpl
	// for use with ChildAt().
	NumChildren() int
	// doWalkTarget should be used sparingly, since it requires
	// additional cast operations. Prefer the type-specific methods instead.
	doWalkTarget(parent TargetContext, fn TargetWalkerFn) (_ TargetImpl, dirty bool, err error)
}

type TargetWalkerFn func(ctx TargetContext, x TargetImpl) (d TargetDecision)

type TargetContext struct {
	depth int
	limit int
}

// Continue is the default decision and returns a zero-value TargetDecision.
// This method exists mainly to improve readability. Implementations of
// TargetWalkerFn can choose to use named return variables and simply
// return.
func (c TargetContext) Continue() TargetDecision {
	return TargetDecision{}
}

// Error will stop the visiting process, unwind the stack, and
// return the given error at the top-level WalkTarget() function.
func (TargetContext) Error(err error) TargetDecision {
	return TargetDecision{err: err}
}

// Halt will stop the visitor and unwind the stack.
func (TargetContext) Halt() TargetDecision {
	return TargetDecision{halt: true}
}

// Limit will visit the children of the current object, but will
// limit the maximum descent depth of the children. A limit of 0
// would skip the current object's children altogether.
func (TargetContext) Limit(limit int) TargetDecision {
	return TargetDecision{depth: limit, limit: true}
}

// Pop will unwind the stack by the specified number of levels and
// continue with the resulting ancestor's next sibling.
func (TargetContext) Pop(levels int) TargetDecision {
	return TargetDecision{depth: levels, pop: true}
}

// TargetDecision allows a TargetWalkerFn to implement flow control.
// Instances of TargetDecision should be obtained from
// TargetContext, however the zero value for this type is "continue".
// A TargetDecision may be further customized with various side-effects.
type TargetDecision struct {
	depth   int
	err     error
	halt    bool
	limit   bool
	pop     bool
	post    TargetWalkerFn
	replace TargetImpl
}

// Post will cause the given function to be executed after any
// children of the current object have been visited.
func (d TargetDecision) Post(fn TargetWalkerFn) TargetDecision {
	d.post = fn
	return d
}

// Replace will replace the value being visited with the replacement.
// The value must be assignable to the field, slice element, etc. that
// holds the visited value or the generated code will panic.
func (d TargetDecision) Replace(replacement TargetImpl) TargetDecision {
	d.replace = replacement
	return d
}

var targetHaltErr = errors.New("halt")

func (x *ByRefType) ChildAt(index int) TargetImpl {
	switch index {
	default:
		panic(fmt.Errorf("child index out of range: %d", index))
	}
}

func (x *ByRefType) ChildNamed(name string) (_ TargetImpl, ok bool) {
	switch name {
	default:
		return nil, false
	}
}

// TargetKind returns TargetIsByRefType.
func (x *ByRefType) TargetKind() TargetKind {
	return TargetIsByRefType
}

// NumChildren returns 0.
func (x *ByRefType) NumChildren() int {
	return 0
}

func (x *ByRefType) WalkTarget(fn TargetWalkerFn) (next *ByRefType, dirty bool, err error) {
	next, dirty, err = x.doWalkTargetImpl(TargetContext{}, fn)
	if err == targetHaltErr {
		err = nil
	} else if err != nil {
		next = nil
		dirty = false
	}
	return
}

func (x *ByRefType) doWalkTarget(parent TargetContext, fn TargetWalkerFn) (ret TargetImpl, dirty bool, err error) {
	return x.doWalkTargetImpl(parent, fn)
}

func (x *ByRefType) doWalkTargetImpl(parent TargetContext, fn TargetWalkerFn) (ret *ByRefType, dirty bool, err error) {
	ctx := parent
	ctx.depth++
	ret = x

	if ctx.limit > 0 && ctx.depth == ctx.limit {
		return
	}

	d := fn(ctx, ret)
	if d.err != nil {
		err = d.err
		return
	}
	if d.replace != nil {
		ret = d.replace.(*ByRefType)
		dirty = true
	}
	if d.halt {
		err = targetHaltErr
		return
	}

	if d.post != nil {
		d = d.post(ctx, ret)
		if d.err != nil {
			err = d.err
			return
		}
		if d.replace != nil {
			ret = d.replace.(*ByRefType)
		}
		if d.halt {
			err = targetHaltErr
			return
		}
	}

	return
}
func (x *ByValType) ChildAt(index int) TargetImpl {
	switch index {
	default:
		panic(fmt.Errorf("child index out of range: %d", index))
	}
}

func (x *ByValType) ChildNamed(name string) (_ TargetImpl, ok bool) {
	switch name {
	default:
		return nil, false
	}
}

// TargetKind returns TargetIsByValType.
func (x *ByValType) TargetKind() TargetKind {
	return TargetIsByValType
}

// NumChildren returns 0.
func (x *ByValType) NumChildren() int {
	return 0
}

func (x *ByValType) WalkTarget(fn TargetWalkerFn) (next *ByValType, dirty bool, err error) {
	next, dirty, err = x.doWalkTargetImpl(TargetContext{}, fn)
	if err == targetHaltErr {
		err = nil
	} else if err != nil {
		next = nil
		dirty = false
	}
	return
}

func (x *ByValType) doWalkTarget(parent TargetContext, fn TargetWalkerFn) (ret TargetImpl, dirty bool, err error) {
	return x.doWalkTargetImpl(parent, fn)
}

func (x *ByValType) doWalkTargetImpl(parent TargetContext, fn TargetWalkerFn) (ret *ByValType, dirty bool, err error) {
	ctx := parent
	ctx.depth++
	ret = x

	if ctx.limit > 0 && ctx.depth == ctx.limit {
		return
	}

	d := fn(ctx, ret)
	if d.err != nil {
		err = d.err
		return
	}
	if d.replace != nil {
		ret = d.replace.(*ByValType)
		dirty = true
	}
	if d.halt {
		err = targetHaltErr
		return
	}

	if d.post != nil {
		d = d.post(ctx, ret)
		if d.err != nil {
			err = d.err
			return
		}
		if d.replace != nil {
			ret = d.replace.(*ByValType)
		}
		if d.halt {
			err = targetHaltErr
			return
		}
	}

	return
}
func (x *ContainerType) ChildAt(index int) TargetImpl {
	switch index {
	case 0:
		return &x.ByRef
	case 1:
		if e := x.ByRefPtr; e == nil {
			return nil
		} else {
			return e
		}
	case 2:
		if e := x.ByRefSlice; e == nil {
			return nil
		} else {
			return ByRefTypeSlice(e)
		}
	case 3:
		if e := x.ByRefPtrSlice; e == nil {
			return nil
		} else {
			return ByRefTypePtrSlice(e)
		}
	case 4:
		return &x.ByVal
	case 5:
		if e := x.ByValPtr; e == nil {
			return nil
		} else {
			return e
		}
	case 6:
		if e := x.ByValSlice; e == nil {
			return nil
		} else {
			return ByValTypeSlice(e)
		}
	case 7:
		if e := x.ByValPtrSlice; e == nil {
			return nil
		} else {
			return ByValTypePtrSlice(e)
		}
	case 8:
		if e := x.Container; e == nil {
			return nil
		} else {
			return e
		}
	case 9:
		if e := x.AnotherTarget; e == nil {
			return nil
		} else {
			return convertTargetToTargetImpl(e)
		}
	case 10:
		if e := x.AnotherTargetPtr; e == nil {
			return nil
		} else {
			return convertTargetToTargetImpl(*e)
		}
	case 11:
		if e := x.EmbedsTarget; e == nil {
			return nil
		} else {
			return convertEmbedsTargetToTargetImpl(e)
		}
	case 12:
		if e := x.EmbedsTargetPtr; e == nil {
			return nil
		} else {
			return convertEmbedsTargetToTargetImpl(*e)
		}
	case 13:
		if e := x.TargetSlice; e == nil {
			return nil
		} else {
			return TargetSlice(e)
		}
	case 14:
		if e := x.NamedTargets; e == nil {
			return nil
		} else {
			return Targets(e)
		}
	case 15:
		if e := x.InterfacePtrSlice; e == nil {
			return nil
		} else {
			return TargetPtrSlice(e)
		}
	default:
		panic(fmt.Errorf("child index out of range: %d", index))
	}
}

func (x *ContainerType) ChildNamed(name string) (_ TargetImpl, ok bool) {
	switch name {
	case "ByRef":
		return &x.ByRef, true
	case "ByRefPtr":
		return x.ByRefPtr, true
	case "ByRefSlice":
		return ByRefTypeSlice(x.ByRefSlice), true
	case "ByRefPtrSlice":
		return ByRefTypePtrSlice(x.ByRefPtrSlice), true
	case "ByVal":
		return &x.ByVal, true
	case "ByValPtr":
		return x.ByValPtr, true
	case "ByValSlice":
		return ByValTypeSlice(x.ByValSlice), true
	case "ByValPtrSlice":
		return ByValTypePtrSlice(x.ByValPtrSlice), true
	case "Container":
		return x.Container, true
	case "AnotherTarget":
		return convertTargetToTargetImpl(x.AnotherTarget), true
	case "AnotherTargetPtr":
		if e := x.AnotherTargetPtr; e == nil {
			return nil, false
		} else {
			return convertTargetToTargetImpl(*e), true
		}
	case "EmbedsTarget":
		return convertEmbedsTargetToTargetImpl(x.EmbedsTarget), true
	case "EmbedsTargetPtr":
		if e := x.EmbedsTargetPtr; e == nil {
			return nil, false
		} else {
			return convertEmbedsTargetToTargetImpl(*e), true
		}
	case "TargetSlice":
		return TargetSlice(x.TargetSlice), true
	case "NamedTargets":
		return Targets(x.NamedTargets), true
	case "InterfacePtrSlice":
		return TargetPtrSlice(x.InterfacePtrSlice), true
	default:
		return nil, false
	}
}

// TargetKind returns TargetIsContainerType.
func (x *ContainerType) TargetKind() TargetKind {
	return TargetIsContainerType
}

// NumChildren returns 16.
func (x *ContainerType) NumChildren() int {
	return 16
}

func (x *ContainerType) WalkTarget(fn TargetWalkerFn) (next *ContainerType, dirty bool, err error) {
	next, dirty, err = x.doWalkTargetImpl(TargetContext{}, fn)
	if err == targetHaltErr {
		err = nil
	} else if err != nil {
		next = nil
		dirty = false
	}
	return
}

func (x *ContainerType) doWalkTarget(parent TargetContext, fn TargetWalkerFn) (ret TargetImpl, dirty bool, err error) {
	return x.doWalkTargetImpl(parent, fn)
}

func (x *ContainerType) doWalkTargetImpl(parent TargetContext, fn TargetWalkerFn) (ret *ContainerType, dirty bool, err error) {
	ctx := parent
	ctx.depth++
	ret = x

	if ctx.limit > 0 && ctx.depth == ctx.limit {
		return
	}

	d := fn(ctx, ret)
	if d.err != nil {
		err = d.err
		return
	}
	if d.replace != nil {
		ret = d.replace.(*ContainerType)
		dirty = true
	}
	if d.halt {
		err = targetHaltErr
		return
	}

	newByRef := ret.ByRef
	newByRefPtr := ret.ByRefPtr
	newByRefSlice := ret.ByRefSlice
	newByRefPtrSlice := ret.ByRefPtrSlice
	newByVal := ret.ByVal
	newByValPtr := ret.ByValPtr
	newByValSlice := ret.ByValSlice
	newByValPtrSlice := ret.ByValPtrSlice
	newContainer := ret.Container
	newAnotherTarget := ret.AnotherTarget
	newAnotherTargetPtr := ret.AnotherTargetPtr
	newEmbedsTarget := ret.EmbedsTarget
	newEmbedsTargetPtr := ret.EmbedsTargetPtr
	newTargetSlice := ret.TargetSlice
	newNamedTargets := ret.NamedTargets
	newInterfacePtrSlice := ret.InterfacePtrSlice

	fieldChanged := false

	{
		y, d, e := (ret.ByRef).doWalkTargetImpl(ctx, fn)
		if d {
			fieldChanged = true
			newByRef = *y
		}
		if e != nil {
			err = e
			if e == targetHaltErr {
				goto halting
			} else {
				return
			}
		}
	}
	if ret.ByRefPtr != nil {
		y, d, e := (ret.ByRefPtr).doWalkTargetImpl(ctx, fn)
		if d {
			fieldChanged = true
			newByRefPtr = y
		}
		if e != nil {
			err = e
			if e == targetHaltErr {
				goto halting
			} else {
				return
			}
		}
	}
	if ret.ByRefSlice != nil {
		y, d, e := (ByRefTypeSlice(ret.ByRefSlice)).doWalkTargetImpl(ctx, fn)
		if d {
			fieldChanged = true
			newByRefSlice = ByRefTypeSlice(y)
		}
		if e != nil {
			err = e
			if e == targetHaltErr {
				goto halting
			} else {
				return
			}
		}
	}
	if ret.ByRefPtrSlice != nil {
		y, d, e := (ByRefTypePtrSlice(ret.ByRefPtrSlice)).doWalkTargetImpl(ctx, fn)
		if d {
			fieldChanged = true
			newByRefPtrSlice = ByRefTypePtrSlice(y)
		}
		if e != nil {
			err = e
			if e == targetHaltErr {
				goto halting
			} else {
				return
			}
		}
	}
	{
		y, d, e := (ret.ByVal).doWalkTargetImpl(ctx, fn)
		if d {
			fieldChanged = true
			newByVal = *y
		}
		if e != nil {
			err = e
			if e == targetHaltErr {
				goto halting
			} else {
				return
			}
		}
	}
	if ret.ByValPtr != nil {
		y, d, e := (ret.ByValPtr).doWalkTargetImpl(ctx, fn)
		if d {
			fieldChanged = true
			newByValPtr = y
		}
		if e != nil {
			err = e
			if e == targetHaltErr {
				goto halting
			} else {
				return
			}
		}
	}
	if ret.ByValSlice != nil {
		y, d, e := (ByValTypeSlice(ret.ByValSlice)).doWalkTargetImpl(ctx, fn)
		if d {
			fieldChanged = true
			newByValSlice = ByValTypeSlice(y)
		}
		if e != nil {
			err = e
			if e == targetHaltErr {
				goto halting
			} else {
				return
			}
		}
	}
	if ret.ByValPtrSlice != nil {
		y, d, e := (ByValTypePtrSlice(ret.ByValPtrSlice)).doWalkTargetImpl(ctx, fn)
		if d {
			fieldChanged = true
			newByValPtrSlice = ByValTypePtrSlice(y)
		}
		if e != nil {
			err = e
			if e == targetHaltErr {
				goto halting
			} else {
				return
			}
		}
	}
	if ret.Container != nil {
		y, d, e := (ret.Container).doWalkTargetImpl(ctx, fn)
		if d {
			fieldChanged = true
			newContainer = y
		}
		if e != nil {
			err = e
			if e == targetHaltErr {
				goto halting
			} else {
				return
			}
		}
	}
	if ret.AnotherTarget != nil {
		y, d, e := (convertTargetToTargetImpl(ret.AnotherTarget)).doWalkTarget(ctx, fn)
		if d {
			fieldChanged = true
			newAnotherTarget = y.(Target)
		}
		if e != nil {
			err = e
			if e == targetHaltErr {
				goto halting
			} else {
				return
			}
		}
	}
	if ret.AnotherTargetPtr != nil {
		y, d, e := (convertTargetToTargetImpl(*ret.AnotherTargetPtr)).doWalkTarget(ctx, fn)
		if d {
			fieldChanged = true
			yy := y.(Target)
			newAnotherTargetPtr = &yy
		}
		if e != nil {
			err = e
			if e == targetHaltErr {
				goto halting
			} else {
				return
			}
		}
	}
	if ret.EmbedsTarget != nil {
		y, d, e := (convertEmbedsTargetToTargetImpl(ret.EmbedsTarget)).doWalkTarget(ctx, fn)
		if d {
			fieldChanged = true
			newEmbedsTarget = y.(EmbedsTarget)
		}
		if e != nil {
			err = e
			if e == targetHaltErr {
				goto halting
			} else {
				return
			}
		}
	}
	if ret.EmbedsTargetPtr != nil {
		y, d, e := (convertEmbedsTargetToTargetImpl(*ret.EmbedsTargetPtr)).doWalkTarget(ctx, fn)
		if d {
			fieldChanged = true
			yy := y.(EmbedsTarget)
			newEmbedsTargetPtr = &yy
		}
		if e != nil {
			err = e
			if e == targetHaltErr {
				goto halting
			} else {
				return
			}
		}
	}
	if ret.TargetSlice != nil {
		y, d, e := (TargetSlice(ret.TargetSlice)).doWalkTargetImpl(ctx, fn)
		if d {
			fieldChanged = true
			newTargetSlice = TargetSlice(y)
		}
		if e != nil {
			err = e
			if e == targetHaltErr {
				goto halting
			} else {
				return
			}
		}
	}
	if ret.NamedTargets != nil {
		y, d, e := (ret.NamedTargets).doWalkTargetImpl(ctx, fn)
		if d {
			fieldChanged = true
			newNamedTargets = Targets(y)
		}
		if e != nil {
			err = e
			if e == targetHaltErr {
				goto halting
			} else {
				return
			}
		}
	}
	if ret.InterfacePtrSlice != nil {
		y, d, e := (TargetPtrSlice(ret.InterfacePtrSlice)).doWalkTargetImpl(ctx, fn)
		if d {
			fieldChanged = true
			newInterfacePtrSlice = TargetPtrSlice(y)
		}
		if e != nil {
			err = e
			if e == targetHaltErr {
				goto halting
			} else {
				return
			}
		}
	}

halting:
	if fieldChanged {
		dirty = true
		ret = &ContainerType{
			ByRef:             newByRef,
			ByRefPtr:          newByRefPtr,
			ByRefSlice:        newByRefSlice,
			ByRefPtrSlice:     newByRefPtrSlice,
			ByVal:             newByVal,
			ByValPtr:          newByValPtr,
			ByValSlice:        newByValSlice,
			ByValPtrSlice:     newByValPtrSlice,
			Container:         newContainer,
			AnotherTarget:     newAnotherTarget,
			AnotherTargetPtr:  newAnotherTargetPtr,
			EmbedsTarget:      newEmbedsTarget,
			EmbedsTargetPtr:   newEmbedsTargetPtr,
			TargetSlice:       newTargetSlice,
			NamedTargets:      newNamedTargets,
			InterfacePtrSlice: newInterfacePtrSlice,
			ignored:           ret.ignored,
			Ignored:           ret.Ignored}
	}

	if d.post != nil {
		d = d.post(ctx, ret)
		if d.err != nil {
			err = d.err
			return
		}
		if d.replace != nil {
			ret = d.replace.(*ContainerType)
		}
		if d.halt {
			err = targetHaltErr
			return
		}
	}

	return
}

type ByRefTypePtrSlice []*ByRefType

// ChildAt returns the N-th element of the slice. It will panic if the
// provided index is out-of-bounds.
func (x ByRefTypePtrSlice) ChildAt(index int) TargetImpl { return x[index] }

// ChildNamed always returns nil, false for a slice type.
func (x ByRefTypePtrSlice) ChildNamed(name string) (_ TargetImpl, ok bool) {
	return nil, false
}

// Kind returns TargetIsByRefTypeSlice.
func (x ByRefTypePtrSlice) TargetKind() TargetKind {
	return TargetIsByRefTypeSlice
}

// NumChildren returns the length of the slice.
func (x ByRefTypePtrSlice) NumChildren() int {
	return len(x)
}

func (x ByRefTypePtrSlice) doWalkTarget(parent TargetContext, fn TargetWalkerFn) (_ TargetImpl, dirty bool, err error) {
	return x.doWalkTargetImpl(parent, fn)
}

func (x ByRefTypePtrSlice) doWalkTargetImpl(parent TargetContext, fn TargetWalkerFn) (ret ByRefTypePtrSlice, dirty bool, err error) {
	if x == nil {
		return nil, false, nil
	}
	for i := range x {
		if x[i] != nil {
			z, d, e := (x[i]).doWalkTargetImpl(parent, fn)
			if d {
				dirty = true

				if ret == nil {
					ret = make(ByRefTypePtrSlice, len(x))
					copy(ret, x[:i])
				}

				ret[i] = z
			}
			if e != nil {
				err = e
				if e == targetHaltErr {
					goto halting
				} else {
					return
				}
			}
		}
	}
halting:
	if ret == nil {
		ret = x
	}
	return
}

type ByValTypePtrSlice []*ByValType

// ChildAt returns the N-th element of the slice. It will panic if the
// provided index is out-of-bounds.
func (x ByValTypePtrSlice) ChildAt(index int) TargetImpl { return x[index] }

// ChildNamed always returns nil, false for a slice type.
func (x ByValTypePtrSlice) ChildNamed(name string) (_ TargetImpl, ok bool) {
	return nil, false
}

// Kind returns TargetIsByValTypeSlice.
func (x ByValTypePtrSlice) TargetKind() TargetKind {
	return TargetIsByValTypeSlice
}

// NumChildren returns the length of the slice.
func (x ByValTypePtrSlice) NumChildren() int {
	return len(x)
}

func (x ByValTypePtrSlice) doWalkTarget(parent TargetContext, fn TargetWalkerFn) (_ TargetImpl, dirty bool, err error) {
	return x.doWalkTargetImpl(parent, fn)
}

func (x ByValTypePtrSlice) doWalkTargetImpl(parent TargetContext, fn TargetWalkerFn) (ret ByValTypePtrSlice, dirty bool, err error) {
	if x == nil {
		return nil, false, nil
	}
	for i := range x {
		if x[i] != nil {
			z, d, e := (x[i]).doWalkTargetImpl(parent, fn)
			if d {
				dirty = true

				if ret == nil {
					ret = make(ByValTypePtrSlice, len(x))
					copy(ret, x[:i])
				}

				ret[i] = z
			}
			if e != nil {
				err = e
				if e == targetHaltErr {
					goto halting
				} else {
					return
				}
			}
		}
	}
halting:
	if ret == nil {
		ret = x
	}
	return
}

type TargetPtrSlice []*Target

// ChildAt returns the N-th element of the slice. It will panic if the
// provided index is out-of-bounds.
func (x TargetPtrSlice) ChildAt(index int) TargetImpl {
	if e := x[index]; e == nil {
		return nil
	} else {
		return convertTargetToTargetImpl(*e)
	}
}

// ChildNamed always returns nil, false for a slice type.
func (x TargetPtrSlice) ChildNamed(name string) (_ TargetImpl, ok bool) {
	return nil, false
}

// Kind returns TargetIsTargetSlice.
func (x TargetPtrSlice) TargetKind() TargetKind {
	return TargetIsTargetSlice
}

// NumChildren returns the length of the slice.
func (x TargetPtrSlice) NumChildren() int {
	return len(x)
}

func (x TargetPtrSlice) doWalkTarget(parent TargetContext, fn TargetWalkerFn) (_ TargetImpl, dirty bool, err error) {
	return x.doWalkTargetImpl(parent, fn)
}

func (x TargetPtrSlice) doWalkTargetImpl(parent TargetContext, fn TargetWalkerFn) (ret TargetPtrSlice, dirty bool, err error) {
	if x == nil {
		return nil, false, nil
	}
	for i := range x {
		if x[i] != nil && *x[i] != nil {
			z, d, e := (convertTargetToTargetImpl(*x[i])).doWalkTarget(parent, fn)
			if d {
				dirty = true

				if ret == nil {
					ret = make(TargetPtrSlice, len(x))
					copy(ret, x[:i])
				}

				zz := z.(Target)
				ret[i] = &zz
			}
			if e != nil {
				err = e
				if e == targetHaltErr {
					goto halting
				} else {
					return
				}
			}
		}
	}
halting:
	if ret == nil {
		ret = x
	}
	return
}

type ByRefTypeSlice []ByRefType

// ChildAt returns the N-th element of the slice. It will panic if the
// provided index is out-of-bounds.
func (x ByRefTypeSlice) ChildAt(index int) TargetImpl { return &x[index] }

// ChildNamed always returns nil, false for a slice type.
func (x ByRefTypeSlice) ChildNamed(name string) (_ TargetImpl, ok bool) {
	return nil, false
}

// Kind returns TargetIsByRefTypeSlice.
func (x ByRefTypeSlice) TargetKind() TargetKind {
	return TargetIsByRefTypeSlice
}

// NumChildren returns the length of the slice.
func (x ByRefTypeSlice) NumChildren() int {
	return len(x)
}

func (x ByRefTypeSlice) doWalkTarget(parent TargetContext, fn TargetWalkerFn) (_ TargetImpl, dirty bool, err error) {
	return x.doWalkTargetImpl(parent, fn)
}

func (x ByRefTypeSlice) doWalkTargetImpl(parent TargetContext, fn TargetWalkerFn) (ret ByRefTypeSlice, dirty bool, err error) {
	if x == nil {
		return nil, false, nil
	}
	for i := range x {
		{
			z, d, e := (x[i]).doWalkTargetImpl(parent, fn)
			if d {
				dirty = true

				if ret == nil {
					ret = make(ByRefTypeSlice, len(x))
					copy(ret, x[:i])
				}

				ret[i] = *z
			}
			if e != nil {
				err = e
				if e == targetHaltErr {
					goto halting
				} else {
					return
				}
			}
		}
	}
halting:
	if ret == nil {
		ret = x
	}
	return
}

type ByValTypeSlice []ByValType

// ChildAt returns the N-th element of the slice. It will panic if the
// provided index is out-of-bounds.
func (x ByValTypeSlice) ChildAt(index int) TargetImpl { return &x[index] }

// ChildNamed always returns nil, false for a slice type.
func (x ByValTypeSlice) ChildNamed(name string) (_ TargetImpl, ok bool) {
	return nil, false
}

// Kind returns TargetIsByValTypeSlice.
func (x ByValTypeSlice) TargetKind() TargetKind {
	return TargetIsByValTypeSlice
}

// NumChildren returns the length of the slice.
func (x ByValTypeSlice) NumChildren() int {
	return len(x)
}

func (x ByValTypeSlice) doWalkTarget(parent TargetContext, fn TargetWalkerFn) (_ TargetImpl, dirty bool, err error) {
	return x.doWalkTargetImpl(parent, fn)
}

func (x ByValTypeSlice) doWalkTargetImpl(parent TargetContext, fn TargetWalkerFn) (ret ByValTypeSlice, dirty bool, err error) {
	if x == nil {
		return nil, false, nil
	}
	for i := range x {
		{
			z, d, e := (x[i]).doWalkTargetImpl(parent, fn)
			if d {
				dirty = true

				if ret == nil {
					ret = make(ByValTypeSlice, len(x))
					copy(ret, x[:i])
				}

				ret[i] = *z
			}
			if e != nil {
				err = e
				if e == targetHaltErr {
					goto halting
				} else {
					return
				}
			}
		}
	}
halting:
	if ret == nil {
		ret = x
	}
	return
}

type TargetSlice []Target

// ChildAt returns the N-th element of the slice. It will panic if the
// provided index is out-of-bounds.
func (x TargetSlice) ChildAt(index int) TargetImpl { return convertTargetToTargetImpl(x[index]) }

// ChildNamed always returns nil, false for a slice type.
func (x TargetSlice) ChildNamed(name string) (_ TargetImpl, ok bool) {
	return nil, false
}

// Kind returns TargetIsTargetSlice.
func (x TargetSlice) TargetKind() TargetKind {
	return TargetIsTargetSlice
}

// NumChildren returns the length of the slice.
func (x TargetSlice) NumChildren() int {
	return len(x)
}

func (x TargetSlice) doWalkTarget(parent TargetContext, fn TargetWalkerFn) (_ TargetImpl, dirty bool, err error) {
	return x.doWalkTargetImpl(parent, fn)
}

func (x TargetSlice) doWalkTargetImpl(parent TargetContext, fn TargetWalkerFn) (ret TargetSlice, dirty bool, err error) {
	if x == nil {
		return nil, false, nil
	}
	for i := range x {
		if x[i] != nil {
			z, d, e := (convertTargetToTargetImpl(x[i])).doWalkTarget(parent, fn)
			if d {
				dirty = true

				if ret == nil {
					ret = make(TargetSlice, len(x))
					copy(ret, x[:i])
				}

				ret[i] = z.(Target)
			}
			if e != nil {
				err = e
				if e == targetHaltErr {
					goto halting
				} else {
					return
				}
			}
		}
	}
halting:
	if ret == nil {
		ret = x
	}
	return
}

// ChildAt returns the N-th element of the slice. It will panic if the
// provided index is out-of-bounds.
func (x Targets) ChildAt(index int) TargetImpl { return convertTargetToTargetImpl(x[index]) }

// ChildNamed always returns nil, false for a slice type.
func (x Targets) ChildNamed(name string) (_ TargetImpl, ok bool) {
	return nil, false
}

// Kind returns TargetIsTargetSlice.
func (x Targets) TargetKind() TargetKind {
	return TargetIsTargetSlice
}

// NumChildren returns the length of the slice.
func (x Targets) NumChildren() int {
	return len(x)
}

func (x Targets) doWalkTarget(parent TargetContext, fn TargetWalkerFn) (_ TargetImpl, dirty bool, err error) {
	return x.doWalkTargetImpl(parent, fn)
}

func (x Targets) doWalkTargetImpl(parent TargetContext, fn TargetWalkerFn) (ret Targets, dirty bool, err error) {
	if x == nil {
		return nil, false, nil
	}
	for i := range x {
		if x[i] != nil {
			z, d, e := (convertTargetToTargetImpl(x[i])).doWalkTarget(parent, fn)
			if d {
				dirty = true

				if ret == nil {
					ret = make(Targets, len(x))
					copy(ret, x[:i])
				}

				ret[i] = z.(Target)
			}
			if e != nil {
				err = e
				if e == targetHaltErr {
					goto halting
				} else {
					return
				}
			}
		}
	}
halting:
	if ret == nil {
		ret = x
	}
	return
}

func convertEmbedsTargetToTargetImpl(x EmbedsTarget) TargetImpl {
	switch t := x.(type) {
	case TargetImpl:
		return t
	case ByValType:
		return &t
	default:
		return nil
	}
}

func convertTargetToTargetImpl(x Target) TargetImpl {
	switch t := x.(type) {
	case TargetImpl:
		return t
	case ByValType:
		return &t
	default:
		return nil
	}
}

// TargetKind is a type token.
type TargetKind int

const (
	_ TargetKind = iota
	TargetIsByRefType
	TargetIsByRefTypeSlice
	TargetIsByValType
	TargetIsByValTypeSlice
	TargetIsContainerType
	TargetIsContainerTypeSlice
	TargetIsEmbedsTarget
	TargetIsEmbedsTargetSlice
	TargetIsTarget
	TargetIsTargetSlice
)

// Elem returns a slice's element kind, or the the input kind.
func (k TargetKind) Elem() TargetKind {
	if k.IsSlice() {
		return k - 1
	}
	return k
}

// IsSlice indicates if the kind represents a slice.
func (k TargetKind) IsSlice() bool {
	return k%2 == 0
}

// String is for debugging use only. The names returned here are
// not necessarily usable as actual types names.
func (k TargetKind) String() string {
	switch k {
	case TargetIsByRefType:
		return "ByRefType"
	case TargetIsByRefTypeSlice:
		return "[]ByRefType"
	case TargetIsByValType:
		return "ByValType"
	case TargetIsByValTypeSlice:
		return "[]ByValType"
	case TargetIsContainerType:
		return "ContainerType"
	case TargetIsContainerTypeSlice:
		return "[]ContainerType"
	case TargetIsEmbedsTarget:
		return "EmbedsTarget"
	case TargetIsEmbedsTargetSlice:
		return "[]EmbedsTarget"
	case TargetIsTarget:
		return "Target"
	case TargetIsTargetSlice:
		return "[]Target"
	default:
		return fmt.Sprintf("TargetKind(%d)", k)
	}
}
