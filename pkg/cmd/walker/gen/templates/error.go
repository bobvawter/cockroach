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
	TemplateSources["error"] = `
{{- $v := . -}}
// --------------- Error types -----------------------------------------

// {{ Error $v }} is used to communicate errors that occur during a
// traversal. This type provides access to a snapshot of the stack
// when the error occurred to aid in debugging.
type {{ Error $v }} struct {
	reason error
  stack {{ Locations $v }}
}

var _ error = &{{ Error $v }}{}

// Cause returns the causal error.
func (e *{{ Error $v }}) Cause() error {
	return e.reason
}

// Error implements the error interface.
func (e *{{ Error $v }}) Error() string {
	return e.reason.Error()
}

// Stack returns a snapshot of the visitation stack where the enclosed
// error occurred.
func (e *{{ Error $v }}) Stack() {{ Locations $v }} {
  return e.stack
}

// String is for debugging use only.
func (e *{{ Error $v }}) String() string {
	return fmt.Sprintf("%v at %v", e.reason, e.stack)
}
`
}
