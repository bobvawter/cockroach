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
	TemplateSources["enhance"] = `
{{- $v := . -}}
// ------------------------- Type enhancements -------------------------

// {{ Impl $v }} are the methods that we will add to traversable types.
type {{ Impl $v }} interface {
	// pre calls the relevant PreXYZ method on the visitor.
	pre(ctx {{ Context $v }}, v {{ Visitor $v }} ) (bool, error)
	// post calls the relevant PreXYZ method on the visitor.
	post(ctx {{ Context $v }}, v {{ Visitor $v }} ) error
	// traverse visits the fields within the struct.
	traverse(ctx {{ Context $v }}, v {{ Visitor $v }})
}

// Ensure we implement the {{ Impl $v }} interface.
var (
{{ range $s := $v.Structs -}}_ {{ Impl $v }} = {{ New $s }} {};{{ end }}
)

{{ define "makeContext" -}}
{{- $v := Visitation . -}}
{{- $k := Kind . -}}
	{{- if or (eq $k "struct") (eq $k "interface") -}}
		,newScalar{{ Context $v }}(), newTypeCheck{{ Context $v }}({{ TypeOf . }})
	{{- else if eq $k "pointer" -}}
		,newPointer{{ Context $v }}(){{ template "makeContext" ElemOf . }}
	{{- else if eq $k "slice" -}}
		,newSlice{{ Context $v }}(){{ template "makeContext" ElemOf . }}
	{{- end -}}
{{- end }}

{{ range $s := $v.Structs }}
{{- $t := TypeRef $s -}}
func (x {{ $t }} ) pre(ctx {{ Context $v }}, v {{ Visitor $v }}) (bool, error) {
  return v.Pre{{ $s.Name }} ( ctx, x)
}
func (x {{ $t }} ) post(ctx {{ Context $v }}, v {{ Visitor $v }}) error {
  return v.Post{{ $s.Name }} ( ctx, x)
}
func (x {{ $t }} ) traverse(ctx {{ Context $v }}, v {{ Visitor $v }} ) {
  {{- if $s.Fields -}}
		top := ctx.rawStack().Top()
		dirty := false

		// This code is structured to avoid escaping of the
		// newLocalVariables by not passing them into accept(), but
		// by using an x.FieldReference, instead.
		{{ range $f := $s.Fields }}
			new{{$f.Name}} /* {{ TypeRef $f.Target }} */ := {{ GetAsTypeRef $f (print "x." $f.Name) }}
			{{ if IsNullable $f -}}
				if new{{$f.Name}} != nil {
			{{- else -}}
			  {
			{{- end -}}
				top.Field = "{{ $f.Name }}"
				c := build{{ Context $v }}(ctx{{ template "makeContext" $f.Target }})
	    	if y, changed := c.accept(v, {{ GetAsTypeRef $f (print "x." $f.Name) }}); changed { 
      		dirty = true;
					new{{ .Name }} = y.({{ TypeRef $f.Target }});
				}
				c.close()
			}
	  {{- end }}

		// Clear field for Post().
		top.Field = ""

		if dirty && ctx.CanReplace() {
			ctx.Replace({{ New $s }}{
				{{ range $f := $s.Fields -}}
					{{ $f.Name }}: {{ GetFromTypeRef $f (print "new" .Name )}},
				{{ end }}
			})
		}
	{{- end -}}
}
{{ end }}
`
}
