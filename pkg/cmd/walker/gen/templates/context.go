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
	TemplateSources["context"] = `
{{- $v := . -}}
// --------------------------- Base Context ----------------------------

// A base{{ Context $v }} implements an immutable {{ Context $v }}.
// A single instance will be shared across all derived contexts
// to transmit common state.
type base{{ Context $v }} struct {
	context.Context
	stack	*{{ Stack $v }}
}

var _ {{ Context $v }} = &base{{ Context $v }} {}

func (c *base{{ Context $v }}) accept(v {{ Visitor $v }}, x interface{}) (interface{}, bool) {
	return x, false
}

func (c *base{{ Context $v }}) abort(err error) {
	err = &{{ Error $v }}{ reason: err, stack: c.Stack() }
	panic(err)
}

func (*base{{ Context $v }}) close() {}

func (*base{{ Context $v }}) CanReplace() bool {
	return false
}

func (c *base{{ Context $v }}) Replace(n {{ Intf $v }}) {
	c.abort(errors.New("this context cannot replace"))
}

func (c *base{{ Context $v }}) replace(n {{ Impl $v }}) {
	c.abort(errors.New("this context cannot replace"))
}

func (*base{{ Context $v }}) CanInsertBefore() bool {
	return false
}

func (c *base{{ Context $v }}) InsertBefore(n {{ Intf $v }}) {
	c.abort(errors.New("this context cannot insert"))
}

func (*base{{ Context $v }}) CanInsertAfter() bool {
	return false
}

func (c *base{{ Context $v }}) InsertAfter(n {{ Intf $v }}) {
	c.abort(errors.New("this context cannot insert"))
}

func (*base{{ Context $v }}) CanRemove() bool {
	return false
}

func (c *base{{ Context $v }}) rawStack() *{{ Stack $v }} {
	return c.stack
}

func (c *base{{ Context $v }}) Remove() {
	c.abort(errors.New("this context cannot remove"))
}

func (c *base{{ Context $v }}) Stack() {{ Locations $v }} {
	return c.stack.Copy()
}

func (c *base{{ Context $v }}) StackLen() int {
  return c.stack.Len()
}
`
}
