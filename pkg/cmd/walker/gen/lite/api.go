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

package lite

func init() {
	TemplateSources["50api"] = `
{{- /*gotype: github.com/cockroachdb/cockroach/pkg/cmd/walker/gen.visitation*/ -}}
{{- $v := . -}}
{{- $Context := T $v "Context" -}}
{{- $Decision := T $v "Decision" -}}
{{- $errHalt := t $v "HaltErr" -}}
{{- $Intf := T $v "" -}}
{{- $Impl := T $v "Impl" -}}
{{- $Kind := T $v "Kind" -}}
{{- $WalkerFn := T $v "WalkerFn" -}}

type {{ $Impl }} interface {
	// ChildAt returns the N-th child of an {{ $Impl }}. For struct kinds,
	// this will be the N-th visitable field of the struct. For slices,
	// this will be the N-th element in the slice.
	ChildAt(index int) {{ $Impl }}
	// ChildNamed provides by-name access to fields. This allows types
	// to implement by-convention protocols.
	ChildNamed(name string) (_ {{ $Impl }}, ok bool)
	// {{ $Intf }}Kind returns a type token to identify the {{ $Impl }}.
	{{ $Intf }}Kind() {{ $Intf }}Kind
	// NumChildren returns the number of children in the {{ $Impl }}
	// for use with ChildAt().
	NumChildren() int
	// doWalk{{ $Intf }} should be used sparingly, since it requires
	// additional cast operations. Prefer the type-specific methods instead.
	doWalk{{ $Intf }}(parent {{ $Context }}, fn {{ $WalkerFn }}) (_ {{ $Impl }}, dirty bool, err error) 
}

type {{ $WalkerFn }} func(ctx {{ $Context }}, x {{ $Impl }}) (d {{ $Decision }})

type {{ $Context }} struct {
	depth int
	limit int
}

// Continue is the default decision and returns a zero-value {{ $Decision }}.
// This method exists mainly to improve readability. Implementations of
// {{ $WalkerFn }} can choose to use named return variables and simply
// return.
func (c {{ $Context }}) Continue() {{ $Decision }} {
	return {{ $Decision }}{}
}

// Error will stop the visiting process, unwind the stack, and
// return the given error at the top-level Walk{{ $Intf }}() function.
func ({{ $Context }}) Error(err error) {{ $Decision }} {
	return {{ $Decision }}{err: err}
}

// Halt will stop the visitor and unwind the stack.
func ({{ $Context }}) Halt() {{ $Decision }} {
	return {{ $Decision }}{halt: true}
}

// Limit will visit the children of the current object, but will
// limit the maximum descent depth of the children. A limit of 0
// would skip the current object's children altogether.
func ({{ $Context }}) Limit(limit int) {{ $Decision }} {
	return {{ $Decision }}{depth: limit, limit: true} 
}

// Pop will unwind the stack by the specified number of levels and
// continue with the resulting ancestor's next sibling.
func ({{ $Context }}) Pop(levels int) {{ $Decision }} {
	return {{ $Decision }}{depth: levels, pop: true}
}

// {{ $Decision }} allows a {{ $WalkerFn }} to implement flow control.
// Instances of {{ $Decision }} should be obtained from
// {{ $Context }}, however the zero value for this type is "continue".
// A {{ $Decision }} may be further customized with various side-effects.
type {{ $Decision }} struct {
	depth   int
	err     error
	halt    bool
	limit   bool
  pop     bool
	post    {{ $WalkerFn }}
	replace {{ $Impl }}
}

// Post will cause the given function to be executed after any
// children of the current object have been visited.
func (d {{ $Decision }}) Post(fn {{ $WalkerFn }}) {{ $Decision }} {
	d.post = fn
	return d
}

// Replace will replace the value being visited with the replacement.
// The value must be assignable to the field, slice element, etc. that
// holds the visited value or the generated code will panic.
func (d {{ $Decision }}) Replace(replacement {{ $Impl }}) {{ $Decision }} {
	d.replace = replacement
	return d
}

var {{ $errHalt }} = errors.New("halt")

{{ range $s := $v.Structs }}
{{- $Facade := Facade $s -}}

func (x {{ $Facade }}) ChildAt(index int) {{ $Impl }} {
	switch index {
		{{ range $i, $f := $s.Fields -}}
		case {{ $i }}:
			{{ if IsNullable $f.Target -}}
				if e := x.{{ $f }}; e == nil {
					return nil
				} else {
					return {{ FacadeGet $f.Target "e" }}
				}
			{{ else -}}
				return {{ FacadeGet $f.Target (print "x." $f) }};
			{{- end -}}
		{{- end -}}
	default:
		panic(fmt.Errorf("child index out of range: %d", index))
	}
}

func (x {{ $Facade }}) ChildNamed(name string) (_ {{ $Impl }}, ok bool) {
	switch name {
		{{ range $f := $s.Fields -}}
		case "{{ $f }}":
			{{ if IsDeref $f.Target (Facade $f.Target) -}}
				if e := x.{{ $f }}; e == nil {
					return nil, false
				} else {
					return {{ FacadeGet $f.Target "e" }}, true
				}
			{{ else -}}
				return {{ FacadeGet $f.Target (print "x." $f) }}, true;
			{{- end -}}
		{{- end -}}
	default:
		return nil, false
	}
}

// {{ $Kind }} returns {{ KindOf $s }}.
func (x {{ $Facade }}) {{ $Kind }}() {{ $Kind }} {
	return {{ KindOf $s }}
}

// NumChildren returns {{ len $s.Fields }}.
func (x {{ $Facade }}) NumChildren() int {
	return {{ len $s.Fields }}
}

func (x {{ $Facade }}) Walk{{ $Intf }}(fn {{ $WalkerFn }}) (next {{ $Facade }}, dirty bool, err error) {
	next, dirty, err = x.{{ WalkFunc $s }}({{ $Context }}{}, fn)
	if err == {{ $errHalt }} {
		err = nil
	} else if err != nil {
		next = nil
		dirty = false
	}
	return
}

func (x {{ $Facade }}) doWalk{{ $Intf }}(parent {{ $Context }}, fn {{ $WalkerFn }}) (ret {{ $Impl }}, dirty bool, err error) {
	return x.{{ WalkFunc $s }}(parent, fn)
}

func (x {{ $Facade }}) {{ WalkFunc $s }}(parent {{ $Context }}, fn {{ $WalkerFn }}) (ret {{ $Facade }}, dirty bool, err error) {
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
		ret = {{ ImplTo $Facade "d.replace" }}
		dirty = true
	}
	if d.halt {
		err = {{ $errHalt }}
		return
	}

	{{ if $s.Fields }}

	{{ range $f := $s.Fields }}
	new{{ $f }} := ret.{{ $f }}
	{{- end }}

	fieldChanged := false

	{{ range $f := $s.Fields }}
	{{ if IsNullable $f.Target }}if ret.{{ $f }} != nil{{ end }} {
	y, d, e := ({{ ImplGet $f.Target (print "ret." $f) }}).{{ WalkFunc $f.Target }}(ctx, fn);
	if d {
		fieldChanged = true
		{{ if IsDeref $f.Target (Facade $f.Target) -}}
		yy := {{ ImplTo $f.Target.Elem "y"}}
		new{{ $f }} = &yy;
		{{ else -}}
		new{{ $f }} = {{ Convert (Facade $f.Target) $f.Target "y" }};
		{{- end -}}
	}
	if e != nil {
		err = e
		if e == {{ $errHalt }} {
			goto halting
		} else {
			return
		}
	}
	};	{{ end }}

halting:
	if fieldChanged {
		dirty = true
		ret = {{ New $s }}{
			{{- range $f := $s.Fields }}
				{{ $f }}: new{{ $f }},
			{{- end -}}
			{{- range $f := $s.OpaqueFields }}
				{{ $f }}: ret.{{ $f }},
			{{- end -}}
		}
	}
	{{ end }}
	
	if d.post != nil {
		d = d.post(ctx, ret)
		if d.err != nil {
			err = d.err
			return
		}
		if d.replace != nil {
			ret = d.replace.({{ $Facade }})
		}
		if d.halt {
			err = {{ $errHalt }}
			return
		}
	}

	return
}
{{ end }}

{{ range $s := $v.Slices }}
{{- $Facade := Facade $s }}
{{ if $s.Synthetic }}
type {{ $Facade }} []{{ $s.Elem }};
{{ end }}

// ChildAt returns the N-th element of the slice. It will panic if the
// provided index is out-of-bounds.
func (x {{ $Facade }}) ChildAt(index int) {{ $Impl }} {
	{{- if IsDeref $s.Elem (Facade $s.Elem) -}}
		if e := x[index]; e == nil {
			return nil
		} else {
			return {{ FacadeGet $s.Elem "e" }};
		}
	{{- else -}}
		return {{ FacadeGet $s.Elem "x[index]" }};
	{{- end -}}
}

// ChildNamed always returns nil, false for a slice type.
func (x {{ $Facade }}) ChildNamed(name string) (_ {{ $Impl }}, ok bool) {
  return nil, false
}

// Kind returns {{ KindOf $s }}.
func (x {{ $Facade }}) {{ $Kind }}() {{ $Kind }} {
	return {{ KindOf $s }}
}

// NumChildren returns the length of the slice.
func (x {{ $Facade }}) NumChildren() int {
	return len(x)
}

func (x {{ $Facade }}) doWalk{{ $Intf }}(parent {{ $Context }}, fn {{ $WalkerFn }}) (_ {{ $Impl }}, dirty bool, err error) {
	return x.{{ WalkFunc $s }}(parent, fn)
}

func (x {{ $Facade }}) {{ WalkFunc $s }}(parent {{ $Context }}, fn {{ $WalkerFn }}) (ret {{ $Facade }}, dirty bool, err error) {
	if x == nil {
		return nil, false, nil
	}
	for i := range x {
		{{ if IsNullable $s.Elem }}if x[i] != nil{{ end }} {{ if IsPointer $s.Elem }}{{ if IsNullable $s.Elem.Elem }} && *x[i] != nil {{ end }}{{ end }}{
			z, d, e := ({{ ImplGet $s.Elem "x[i]" }}).{{ WalkFunc $s.Elem }}(parent, fn);
			if d {
				dirty = true

				if ret == nil {
					ret = make({{ $Facade }}, len(x))
					copy(ret, x[:i])
				}

				{{ if IsDeref $s.Elem (Facade $s.Elem) -}}
					zz := {{ ImplTo $s.Elem.Elem "z"}}
					ret[i] = &zz;
				{{ else -}}
					ret[i] = {{ Convert (Facade $s.Elem) $s.Elem "z" }};
				{{- end -}}
			}
			if e != nil {
				err = e
				if e == {{ $errHalt }} {
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
{{ end }}

{{ range $s := $v.Intfs }}
func convert{{ $s }}To{{ $Impl }}(x {{ $s }}) {{ $Impl }} {
	switch t := x.(type) {
		case {{ $Impl }}:
			return t
		{{ range StructImplsOf $s }}case {{ .}}: return &t;{{ end }}
	default:
		return nil
	}
}
{{ end }}

// {{ $Intf }}Kind is a type token.
type {{ $Intf }}Kind int

const (
	_ {{ $Kind }} = iota
	{{range $s := $v.Structs }}{{ KindOf $s }}; {{ KindOf $s }}Slice;{{ end }}
	{{range $s := $v.Intfs }}{{ KindOf $s }}; {{ KindOf $s }}Slice;{{ end }}
)

// Elem returns a slice's element kind, or the the input kind.
func (k {{ $Kind }}) Elem() {{ $Kind }} {
	if k.IsSlice() {
		return k - 1
	}
	return k
}

// IsSlice indicates if the kind represents a slice.
func (k {{ $Kind }}) IsSlice() bool {
	return k % 2 == 0
}

// String is for debugging use only. The names returned here are
// not necessarily usable as actual types names. 
func (k {{ $Kind }}) String() string {
	switch k {
	{{range $s := $v.Structs -}}
		case {{ KindOf $s }}: return "{{ $s }}";
		case {{ KindOf $s }}Slice: return "[]{{ $s }}";
	{{- end -}}
	{{- range $s := $v.Intfs -}}
		case {{ KindOf $s }}: return "{{ $s }}";
		case {{ KindOf $s }}Slice: return "[]{{ $s }}";
	{{- end -}}
	default:
		return fmt.Sprintf("{{ $Kind }}(%d)", k) 
	}
}

`
}
