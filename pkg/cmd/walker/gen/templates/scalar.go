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
	TemplateSources["scalar"] = `
{{- $v := . -}}
// ------------------------ Scalar Context -----------------------------

// scalar{{ Context $v }} instances should be obtained through
// {{Context $v}}.scalarContext().
type scalar{{ Context $v }} struct {
	{{ Context $v }}
	dirty           bool
	elementContext  {{ Context $v }}
	replacement     interface{}
}

var _ {{ Context $v }} = &scalar{{ Context $v }}{}

func (c *scalar{{ Context $v }}) accept(
	v {{ Visitor $v }}, val interface{},
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

func (c *scalar{{ Context $v }}) close() {
	c.elementContext.close()
	*c = scalar{{ Context $v }}{}
	scalar{{ Context $v }}Pool.Put(c)
}

func (*scalar{{ Context $v }}) CanReplace() bool {
	return true
}
func (c *scalar{{ Context $v }}) Replace(n {{ Intf $v }}) {
	c.dirty = true
	c.replacement = n.({{ Impl $v }})
}
`
}
