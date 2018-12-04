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
	TemplateSources["stack"] = `
{{- $v := . -}}
// ------------------------ Stack Support ------------------------------

// {{ Stack $v }} is a datastructure that's intended to be shared across
// derived instances of {{ Context $v }} to allow the backing array
// to be reused.
type {{ Stack $v }} {{ Locations $v }}


func new{{ Stack $v }}() *{{ Stack $v }} {
	s := make(stackTarget, 0, {{ DefaultStackSize }})
	return &s
}

// Copy duplicates the stack.
func (s *{{ Stack $v }}) Copy() {{ Locations $v }} {
	return {{ Locations $v}}(append((*s)[:0:0], *s...))
}

// Push adds a new location to the stack.
func (s *{{ Stack $v }}) Push(loc {{ Location $v }}) {
	*s = append(*s, loc)
}

// Pop removes the top element from the stack.  It will panic if 
// an empty stack is popped.
func (s *{{ Stack $v }}) Pop() {
	*s = (*s)[:len(*s) - 1]
}

// Len returns the length of the stack.
func (s *{{ Stack $v }}) Len() int {
	return len(*s)
}

// Reset will zero out the stack, retaining the backing array if
// it hasn't grown beyond the default size.
func (s *{{ Stack $v }}) Reset() *{{ Stack $v }} {
	*s = (*s)[:0:{{ DefaultStackSize }}]
	return s
}

// Top returns a pointer to the top frame of the stack so that it may
// be modified in-place.
func (s *{{ Stack $v}}) Top() *{{ Location $v }} {
	return &((*s)[len(*s)-1])
}
`
}
