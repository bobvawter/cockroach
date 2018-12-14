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

package templates

func init() {
	TemplateSources["api"] = `
{{- $v := . -}}
// --------------- Public API ------------------------------------------

// {{ Context $v }} allows for in-place structural modification by a
// {{ Visitor $v }}.
type {{ Context $v }} interface {
	context.Context

	// CanReplace indicates whether or not a call to Replace() can succeed.
	CanReplace() bool
	// Replace will substitute the given value for the value being visited.
	Replace(x {{ Intf $v }})

	// CanInsertBefore indicates whether or not a call to InsertBefore() can succeed.
	CanInsertBefore() bool
	// InsertBefore will insert the given value before the value being visited.
	InsertBefore(x {{ Intf $v }})

	// CanInsertAfter indicates whether or not a call to InsertAfter() can succeed.
	CanInsertAfter() bool
	// InsertAfter will insert the given value after the value being visited.
	// Note that the inserted value will not be traversed by the visitor.
	InsertAfter(x {{ Intf $v }})

	// CanRemove indicates if whether or not a call to Remove() can succeed.
	CanRemove() bool
	// Remove will nullify or delete the value being visited.
	Remove()

	// Stack returns a copy of the objects being visited. This slice
	// is ordered from root to leaf.
	Stack() {{ Locations $v }}
	// StackLen returns the current depth of the stack.
	StackLen() int

	// abort will terminate the processing to emit the given error to the user.
	abort(err error)

	// accept processes a value within the context. This method returns a
	// (possibly unchanged) value and whether or not a change occurred
	// in this value or within a nested context. A context should expect
	// accept() to be called more than once on any given instance.
	accept(v {{ Visitor $v }}, x interface{}) (result interface{}, changed bool)

	// close will cleanup and recycle the context instance. Contexts
	// which enclose other contexts should propagate the call.
	close()

	// rawStack returns the internal stack structure.
	rawStack() *{{ Stack $v }}
}

// {{ Location $v }} is reported by {{ Context $v }}.Stack().
type {{ Location $v }} struct {
	// Value is the object being visited. This will always be a pointer,
	// even for types that are usually visited using by-value semantics.
  Value 		{{ Intf $v }}
  Field			string
	Index			int
}

// String is for debugging use only.
func (l {{ Location $v }}) String() string {
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

type {{ Locations $v }} []{{ Location $v }}

// String is for debugging use only.
func (s {{ Locations $v }}) String() string {
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
// every struct that implements the {{ Intf $v }} interface.
type {{ Visitor $v }} interface {
	{{ range $s := $v.Structs -}}
    Pre{{ $s.Name }}(ctx {{ Context $v }}, x {{ TypeRef $s }} ) (recurse bool, err error)
    Post{{ $s.Name }}(ctx {{ Context $v }}, x {{ TypeRef $s }} ) error
	{{ end }}
}

// A default implementation of {{ Visitor $v }}.
// This has provisions for allowing users to provide default
// pre/post methods.
type {{ Visitor $v }}Base struct {
	DefaultPre  func(ctx {{ Context $v }}, x {{ Intf $v }}) (recurse bool, err error)
	DefaultPost func(ctx {{ Context $v }}, x {{ Intf $v }}) error
}

var _ {{ Visitor $v }} = &{{ Visitor $v }}Base{}

{{ range $s := $v.Structs -}}
// Pre{{ $s.Name }} implements the {{ Visitor $v }} interface.
func (b {{ Visitor $v }}Base ) Pre{{ $s.Name }}(ctx {{ Context $v }}, x {{ TypeRef $s }} ) (recurse bool, err error) {
	if b.DefaultPre == nil {
    return true, nil
  }
  return b.DefaultPre(ctx, x)
}
// Post{{ $s.Name }} implements the {{ Visitor $v }} interface.
func (b {{ Visitor $v }}Base ) Post{{ $s.Name }}(ctx {{ Context $v }}, x {{ TypeRef $s }} ) error {
	if b.DefaultPost == nil {
    return nil
  }
  return b.DefaultPost(ctx, x)
}
{{ end }}
`
}
