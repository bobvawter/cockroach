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

package gen

import (
	"bytes"
	"fmt"
	"go/format"
	"go/types"
	"reflect"
	"sort"
	"strings"
	"text/template"

	templates "github.com/cockroachdb/cockroach/pkg/cmd/walker/gen/lite"
	"github.com/pkg/errors"
)

var allTemplates = make(map[string]*template.Template)

// Register all templates to be generated.
func init() {
	for name, src := range templates.TemplateSources {
		allTemplates[name] = template.Must(template.New(name).Funcs(funcMap).Parse(src))
	}
}

// convert returns an expression that converts expr from one
// type to another.  Not all types are necessarily supported.
func convert(from, to visitableType, expr string) (string, error) {
	for {
		switch tFrom := from.(type) {
		case namedVisitableType:
			// TypeAlias -> Anything
			from = tFrom.Underlying
			continue

		case namedInterfaceType:
			switch tTo := to.(type) {
			case namedInterfaceType:
				// VisitableIntf -> Impl
				if tTo == tTo.v.impl {
					return fmt.Sprintf("convert%sTo%s(%s)", tFrom, tTo, expr), nil
				}
			}
			// VisitableInt -> Anything
			return fmt.Sprintf("%s.(%s)", expr, to), nil

		case namedSliceType:
			switch tTo := to.(type) {
			case namedSliceType:
				// []Foo -> FooSlice
				return fmt.Sprintf("%s(%s)", tTo, expr), nil

			case namedInterfaceType:
				// []Foo -> Intf
				if tFrom.Synthetic == "" && tTo == tTo.Visitation().impl {
					return expr, nil
				} else {
					return fmt.Sprintf("%s(%s)", tFrom.Synthetic, expr), nil
				}
			}

		case namedStruct:
			switch tTo := to.(type) {
			case namedStruct:
				// Struct -> Struct (no-op)
				if tFrom == tTo {
					return expr, nil
				}
			case pointerType:
				// Struct -> *Struct
				if tTo.Elem == tFrom {
					return fmt.Sprintf("&%s", expr), nil
				}
			case namedInterfaceType:
				// Struct -> Intf
				if tTo == tTo.Visitation().impl {
					return expr, nil
				}
			}

		case pointerType:
			// *Something -> *Something
			if from == to {
				return expr, nil
			}
			// *Something -> Something
			if tFrom.Elem == to {
				return fmt.Sprintf("*%s", expr), nil
			}
			switch tFrom.Elem.(type) {
			case namedStruct:
				switch to.(type) {
				case namedInterfaceType:
					// *Struct -> Impl
					if to == to.Visitation().impl {
						return expr, nil
					}
					// *Struct -> SomeOtherInterface
					return fmt.Sprintf("%s.(%s)", expr, to), nil
				}

			case namedInterfaceType:
				switch tTo := to.(type) {
				case namedInterfaceType:
					// *VisitableIntf -> Impl
					if tTo == tTo.v.impl {
						return fmt.Sprintf("convert%sTo%s(*%s)", tFrom.Elem, tTo, expr), nil
					}
				}
			}
		}

		return "", errors.Errorf("unsupported from %s to %s", from, to)
	}
}

func facade(i visitableType) visitableType {
	switch t := i.(type) {
	case namedStruct:
		// Always present structs as pointers.
		return pointerType{t}

	case namedInterfaceType:
		return i.Visitation().impl

	case pointerType:
		switch t.Elem.(type) {
		case namedInterfaceType:
			return i.Visitation().impl
		default:
			return t
		}

	default:
		return t
	}
}

// funcMap contains a map of functions that can be called from within
// the templates. Functions that return types names that we're declaring
// should be parameterized with the name of the traversable interface
// so that multiple traversable interfaces can live in the same package.
var funcMap = template.FuncMap{
	"Convert": convert,
	// DefaultStackSize returns the initial size of the stack slice.
	"DefaultStackSize": func() int { return 32 },
	"ElemOf": func(v visitableType) (ret visitableType, err error) {
		for {
			switch tv := v.(type) {
			case pointerType:
				ret = tv.Elem
			case namedSliceType:
				ret = tv.Elem
			case namedVisitableType:
				v = tv.Underlying
				continue
			default:
				err = errors.Errorf("%+v has no element", v)
			}
			return
		}
	},
	// Error returns the name of an error type.
	"Error": func(v *visitation) string { return v.intfName + "WalkError" },
	// Facade returns the type exposed to the user via generated API.
	"Facade": facade,
	// FacadeGet sugars expr so as to make it assignable to {{ Facade tgt
	// }}. This should be used any time a type is sent to user-facing
	// code. This specifically handles the case where we have a pointer to
	// a by-value type, so that we dereference the pointer, giving the
	// user a consistent type to use in switch statements, casts, etc.
	"FacadeGet": func(tgt visitableType, expr string) (string, error) {
		return convert(tgt, facade(tgt), expr)
	},
	// Intf returns the name of the visitor interface.
	"Intf": func(v *visitation) string { return v.intfName },
	// ImplGet sugars expr so as to make it assignable to {{ $Impl }}.
	// This can be used internally and may be somewhat more efficient
	// than calling FacadeGet, with the caveat that if we have
	// *ByVaueType, it *won't* be dereferenced into ByValueType.
	"ImplGet": func(tgt visitableType, expr string) (string, error) {
		return convert(tgt, tgt.Visitation().impl, expr)
	},
	// ImplTo sugars an expression of type {{ $Impl }} to match
	// the given type.
	"ImplTo": func(tgt visitableType, expr string) (string, error) {
		return convert(tgt.Visitation().impl, tgt, expr)
	},
	// IsDeref is used to allow us to skip field==nil checks.
	// This is cut-down version of convert().
	"IsDeref": func(from, to visitableType) bool {
		for {
			switch tFrom := from.(type) {
			case namedVisitableType:
				// TypeAlias -> Anything
				from = tFrom.Underlying
				continue
			case pointerType:
				switch tTo := to.(type) {
				case namedStruct:
					// *Struct -> Struct
					if tFrom.Elem == tTo {
						return true
					}

				case namedInterfaceType:
					switch tFrom.Elem.(type) {
					case namedInterfaceType:
						// *Intf -> OtherInterface
						return true
					case namedStruct:
						// Special case *SomeStruct -> Impl
						if tTo == tTo.Visitation().impl {
							return false
						}
						return true
					}
				}
			}
			return false
		}
	},
	"KindOf": func(v visitableType) (string, error) {
		suffix := ""
		for {
			switch tv := v.(type) {
			case namedStruct, namedInterfaceType:
				return fmt.Sprintf("%sIs%s%s", tv.Visitation().intfName, tv, suffix), nil
			case namedSliceType:
				v = tv.Elem
				suffix = "Slice" + suffix
			case pointerType:
				v = tv.Elem
			case namedVisitableType:
				v = tv.Underlying
			default:
				return "", errors.Errorf("unsupported type: %s", reflect.TypeOf(tv))
			}
		}
	},
	"IsInterface": func(v visitableType) bool {
		for {
			switch tv := v.(type) {
			case namedVisitableType:
				v = tv.Underlying
			case namedInterfaceType:
				return true
			default:
				return false
			}
		}
	},
	"IsNullable": func(v visitableType) bool {
		for {
			switch tv := v.(type) {
			case namedVisitableType:
				v = tv.Underlying
			case namedInterfaceType, namedSliceType, pointerType:
				return true
			default:
				return false
			}
		}
	},
	"IsSlice": func(v visitableType) bool {
		for {
			switch tv := v.(type) {
			case namedVisitableType:
				v = tv.Underlying
			case namedSliceType:
				return true
			default:
				return false
			}
		}
	},
	"IsPointer": func(v visitableType) bool {
		for {
			switch tv := v.(type) {
			case namedVisitableType:
				v = tv.Underlying
			case pointerType:
				return true
			default:
				return false
			}
		}
	},
	// Location returns the name of our context location type.
	"Location": func(v *visitation) string { return v.intfName + "Location" },
	// Locations returns the name for a slice of Locations.
	"Locations": func(v *visitation) string { return v.intfName + "Locations" },
	"New": func(t namedStruct) string {
		if t.implMode == byRef {
			return "&" + t.String()
		}
		return t.String()
	},
	// Package returns the name of the package we're working in.
	"Package": func(v *visitation) string { return v.pkg.Name() },
	// SourceFile returns the name of the file that defines the interface.
	"SourceFile": func(v *visitation) string {
		return v.gen.fileSet.Position(v.pkg.Scope().Lookup(v.intfName).Pos()).Filename
	},
	// Stack returns the name of the {{ Locations }} stack.
	"Stack": func(v *visitation) string {
		return "stack" + v.intfName
	},
	"StructImplsOf": func(t namedInterfaceType) (ret []namedStruct) {
		for _, s := range t.Visitation().Structs {
			if s.implMode == byValue && types.Implements(s.Named, t.Interface) {
				ret = append(ret, s)
			}
		}
		return
	},
	"t": func(v *visitation, name string) string {
		return fmt.Sprintf("%s%s%s", strings.ToLower(v.intfName[:1]), v.intfName[1:], name)
	},
	"T": func(v *visitation, name string) string {
		return fmt.Sprintf("%s%s", v.intfName, name)
	},
	// TypeOf emits a reference to a reflect.Type token.
	"TypeOf": func(i visitableType) string {
		suffix := ""
		// []*Foo -> FooSlicePtr
		// *[]Foo -> FooPtrSlice
		for {
			switch t := i.(type) {
			case namedSliceType:
				suffix = "Slice" + suffix
				i = t.Elem
			case pointerType:
				suffix = "Ptr" + suffix
				i = t.Elem
			default:
				return fmt.Sprintf("typeOf%s%s", t, suffix)
			}
		}
	},
	// TypeOfIntf emits a reference to a reflect.Type token for the visitable interface.
	"TypeOfIntf": func(v *visitation) string { return "typeOf" + v.intfName },
	// WalkFunc returns the name of the doWalkXYZ method that should be
	// used on the given type.
	"WalkFunc": func(t visitableType) (string, error) {
		for {
			switch tt := t.(type) {
			case namedStruct, namedSliceType:
				return "doWalk" + t.Visitation().intfName + "Impl", nil
			case namedInterfaceType:
				return "doWalk" + t.Visitation().intfName, nil
			case namedVisitableType:
				t = tt.Underlying
			case pointerType:
				t = tt.Elem
			default:
				return "", errors.Errorf("unsupported type %s", reflect.TypeOf(tt))
			}
		}
	},
}

// generateAPI is the main code-generation function. It evaluates
// the embedded template and then calls go/format on the resulting
// code.
func (v *visitation) generateAPI() error {

	// Parse each template and sort the keys.
	sorted := make([]string, 0, len(allTemplates))
	var err error
	for key := range allTemplates {
		sorted = append(sorted, key)
	}
	sort.Strings(sorted)

	// Execute each template in sorted order.
	var buf bytes.Buffer
	for _, key := range sorted {
		if err := allTemplates[key].ExecuteTemplate(&buf, key, v); err != nil {
			return errors.Wrap(err, key)
		}
	}

	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		println(buf.String())
		return err
	}

	outName := strings.ToLower(v.intfName) + "_walker.g"
	if v.inTest {
		outName += "_test"
	}
	outName += ".go"
	out, err := v.gen.writeCloser(outName)
	if err != nil {
		return err
	}

	_, err = out.Write(formatted)
	if x := out.Close(); x != nil && err == nil {
		err = x
	}
	return err
}
