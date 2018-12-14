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
	TemplateSources["slice"] = `
{{- $v := . -}}
// ------------------------ Slice Context ------------------------------

// slice{{ Context $v }} instances should be obtained through
// {{Context $v}}.sliceContext().
type slice{{ Context $v }} struct {
	{{ Context $v }}
	didRemove    			bool
	didReplace   			bool
	dirty        			bool
	elementContext  	{{ Context $v }}
	insertAfter  			[]interface{}
	insertBefore 			[]interface{}
	replacement  			interface{}
}

func (c *slice{{ Context $v }}) accept(
	v {{ Visitor $v }}, val interface{},
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

func (c *slice{{ Context $v }}) close() {
	c.elementContext.close()
	// Nullify any references and push back into pool.
	*c = slice{{ Context $v }}{}
	slice{{ Context $v }}Pool.Put(c)
}

func (c *slice{{ Context $v }}) CanInsertAfter() bool {
	return true
}
func (c *slice{{ Context $v }}) InsertAfter(val {{ Intf $v }}) {
	c.dirty = true
	c.insertAfter = append(c.insertAfter, val)
}
func (c *slice{{ Context $v }}) CanInsertBefore() bool {
	return true
}
func (c *slice{{ Context $v }}) InsertBefore(val {{ Intf $v }}) {
	c.dirty = true
	c.insertBefore = append(c.insertBefore, val)
}
func (c *slice{{ Context $v }}) CanRemove() bool {
	return true
}
func (c *slice{{ Context $v }}) Remove() {
	c.dirty = true
	c.didRemove = true
	c.didReplace = false
}
func (c *slice{{ Context $v }}) CanReplace() bool {
	return true
}
func (c *slice{{ Context $v }}) Replace(x {{ Intf $v }}) {
	c.dirty = true
	c.didRemove = false
	c.didReplace = true
	c.replacement = x
}
`
}
