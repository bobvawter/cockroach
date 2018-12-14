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
	TemplateSources["pointer"] = `
{{- $v := . -}}
// ----------------------- Pointer Context -----------------------------

// pointer{{ Context $v }} is used for impedence matching when we
// have a by-value type in a by-ref field.
type pointer{{ Context $v }} struct {
	{{ Context $v }}
	elementContext {{ Context $v }}
  takeAddr bool
}

var _ {{ Context $v }} = &pointer{{ Context $v }}{}

func (c *pointer{{ Context $v }}) accept(v {{ Visitor $v }}, val interface{}) (interface{}, bool) {
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

func (c *pointer{{ Context $v }}) close() {
	c.elementContext.close()
	*c = pointer{{ Context $v }}{}
	pointer{{ Context $v }}Pool.Put(c)
}
`
}
