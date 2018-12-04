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
	"go/format"
	"sort"
	"strings"
	"text/template"

	"github.com/cockroachdb/cockroach/pkg/cmd/walker/gen/templates"
	"github.com/pkg/errors"
)

var allTemplates = make(map[string]*template.Template)

// Register all templates to be generated.
func init() {
	for name, src := range templates.TemplateSources {
		allTemplates[name] = template.Must(template.New(name).Funcs(funcMap).Parse(src))
	}
}

// funcMap contains a map of functions that can be called from within
// the templates. Functions that return types names that we're declaring
// should be parameterized with the name of the traversable interface
// so that multiple traversable interfaces can live in the same package.
var funcMap = template.FuncMap{
	"AllKnownTypes": func(v *visitation) map[string]visitableType {
		seen := make(map[string]visitableType)

		for _, t := range v.Intfs {
			seen[t.Name()] = t
		}

		for _, t := range v.Structs {
			seen[t.Name()] = t
			for _, f := range t.Fields() {
				seen[f.Target.Name()] = f.Target
				if slice, ok := f.Target.(*sliceType); ok {
					seen[slice.Elem.Name()] = slice.Elem
				}
			}
		}

		return seen
	},
	// Context returns the name of our visitor context type.
	"Context": func(v *visitation) string { return v.intfName + "Context" },
	// DefaultStackSize returns the initial size of the stack slice.
	"DefaultStackSize": func() int { return 32 },
	"ElemOf": func(v visitableType) (ret visitableType, err error) {
		for {
			switch tv := v.(type) {
			case *pointerType:
				ret = tv.Elem
			case *sliceType:
				ret = tv.Elem
			case *namedVisitableType:
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
	// GetAsTypeRef returns a field-access expression which ensures
	// that the field is being read as the type's desired mode.
	"GetAsTypeRef": func(field *fieldInfo, expr string) string {
		switch tgt := field.Target.(type) {
		case *structInfo:
			if tgt.ImplMode == byRef {
				return "&" + expr
			}
		}
		return expr
	},
	// GetFromTypeRef is the inverse of GetAsTypeRef.
	"GetFromTypeRef": func(field *fieldInfo, expr string) string {
		switch tgt := field.Target.(type) {
		case *structInfo:
			if tgt.ImplMode == byRef {
				return "*" + expr
			}
		}
		return expr
	},
	// Impl returns the name of our extended visitable interface.
	"Impl": func(v *visitation) string { return "traversable" + v.intfName },
	// ImplType unwraps the visitableType to extract the underlying,
	// user-defined struct or interface type.
	"ImplType": func(v visitableType) visitableType {
		for {
			switch t := v.(type) {
			case *structInfo:
				return t
			case *namedInterfaceType:
				return t
			case *namedVisitableType:
				v = t.Underlying
			case *pointerType:
				v = t.Elem
			case *sliceType:
				v = t.Elem
			}
		}
	},
	// Intf returns the name of the visitor interface.
	"Intf": func(v *visitation) string { return v.intfName },
	// IsNullable is used to allow us to skip field==nil checks.
	// It accepts fieldInfo or visitableType objects.
	"IsNullable": func(x interface{}) (bool, error) {
		switch thing := x.(type) {
		case *fieldInfo:
			return thing.Target.Mode() == byRef, nil
		case visitableType:
			return thing.Mode() == byRef, nil
		}
		return false, errors.Errorf("unsupported: %+v", x)
	},
	"Kind": func(v visitableType) (string, error) {
		for {
			switch tv := v.(type) {
			case *namedInterfaceType:
				return "interface", nil
			case *namedVisitableType:
				v = tv.Underlying
			case *sliceType:
				return "slice", nil
			case *structInfo:
				return "struct", nil
			case *pointerType:
				// Special-case, we handle a pointer to a by-reference
				// struct as the struct itself.
				if s, ok := tv.Elem.(*structInfo); ok {
					if s.ImplMode == byRef {
						return "struct", nil
					}
				}
				return "pointer", nil
			default:
				return "", errors.Errorf("unsupported kind %+v", v)
			}
		}
	},
	"IsSlice": func(v visitableType) bool {
		for {
			switch tv := v.(type) {
			case *namedVisitableType:
				v = tv.Underlying
			case *sliceType:
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
	// New returns the the syntax to create the given type.
	"New": func(s *structInfo) string {
		if s.ImplMode == byRef {
			return "&" + s.Name()
		}
		return s.Name()
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
	// TypeOf emits a reference to a reflect.Type token.
	"TypeOf": func(i visitableType) string {
		suffix := ""
		// []*Foo -> FooSlicePtr
		// *[]Foo -> FooPtrSlice
		for {
			switch t := i.(type) {
			case *sliceType:
				suffix = "Slice" + suffix
				i = t.Elem
			case *pointerType:
				suffix = "Ptr" + suffix
				i = t.Elem
			default:
				return "typeOf" + t.Name() + suffix
			}
		}
	},
	// TypeOfIntf emits a reference to a reflect.Type token for the visitable interface.
	"TypeOfIntf": func(v *visitation) string { return "typeOf" + v.intfName },
	// TypeRef emits a reference to the given type.  This is the type
	// as it is exposed in the public API.
	"TypeRef": func(typ visitableType) string {
		switch t := typ.(type) {
		case *structInfo:
			if t.ImplMode == byRef {
				return "*" + t.Name()
			}
		}
		return typ.Name()
	},
	// Visitation extracts a pointer to the visitation from the type.
	"Visitation": func(typ visitableType) (*visitation, error) {
		for {
			switch tv := typ.(type) {
			case *structInfo:
				return tv.v, nil
			case *namedInterfaceType:
				return tv.v, nil
			case *pointerType:
				typ = tv.Elem
			case *namedVisitableType:
				typ = tv.Underlying
			case *sliceType:
				typ = tv.Elem
			default:
				return nil, errors.Errorf("unsupported type: %+v", typ)
			}
		}
	},
	// Visitor emits the name of our visitor type.
	"Visitor": func(v *visitation) string { return v.intfName + "Visitor" },
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
