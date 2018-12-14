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
	TemplateSources["walk"] = `
{{- $v := . -}}
// ------------- Walk functions ----------------------------------------

// Pool the root contexts, which own the stack slices.
var root{{ Context $v }}Pool = sync.Pool{New: func() interface{} {
	return &base{{ Context $v }}{stack: new{{ Stack $v }}()}
}}

// walk provides a top-level behavior for setting up a root context
// and for unwinding the stack after a panic.
func walk{{ Intf $v }}(
	ctx context.Context, v {{ Visitor $v }}, tgt {{ Intf $v }}, assignableTo reflect.Type,
) (res {{ Impl $v }}, changed bool, err error) {
	root := root{{ Context $v }}Pool.Get().(*base{{ Context $v }})
	root.Context = ctx

	defer func() {
		// Reset and trim capacity.
		*root = base{{ Context $v }}{stack: root.stack.Reset()}
		root{{ Context $v }}Pool.Put(root)
		if r := recover(); r != nil {
			if we, ok := r.(*{{ Error $v }}); ok {
				err = we
			} else {
				panic(r)
			}
		}
	}()

	c := build{{ Context $v }}(
		root,
		newScalar{{ Context $v }}(),
		newTypeCheck{{ Context $v }}(assignableTo),
	) 
	x, changed := c.accept(v, tgt)
	c.close()
	res = x.({{ Impl $v}})
	return
}

{{ range $i := .Intfs}}
{{- $t := TypeRef $i -}}
// Walk{{ $i.Name }} walks a visitor over an {{ $i.Name }}.
func Walk{{ $i.Name }}(
	ctx context.Context, v {{ Visitor $v }}, tgt {{ $t }},
) (result {{ $t }}, changed bool, err error) {
	t, changed, err := walk{{ Intf $v }}(ctx, v, tgt, {{ TypeOf $i }})
  result = t.({{ $t }})
	return
}
{{ end }}
{{ range $s := .Structs}}
{{- $t := TypeRef $s -}}
// Walk applies the visitor to the {{ $s.Name }}. It returns the original
// value if none of the {{ Context $v }} mutation methods were changed,
// or a replacement value if they were.
func (x {{ $t }} ) Walk(ctx context.Context, v {{ Visitor $v }}) (result {{ $t }}, changed bool, err error) {
	impl, changed, err := walk{{ Intf $v }}(ctx, v, x, {{ TypeOf $s }})
	if err != nil {
    return
  }
  result = impl.( {{ $t }} )
  return
}
{{ end }}
`
}
