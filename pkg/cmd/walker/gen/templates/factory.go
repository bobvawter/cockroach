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
	TemplateSources["factory"] = `
{{- $v := . -}}
// ------------------- Context Factory ---------------------------------


// build{{ Context $v }} sets up the linkages between contexts.
// The "current" context must be the zeroth element, meaning that
// stack must have at least two elements.
func build{{ Context $v }}(stack ...{{ Context $v }}) {{ Context $v }} {
	// Fold the entries together, linking them to their parent context.
	for i, l := 1, len(stack); i<l; i++ {
		switch t := stack[i].(type) {
			case *pointer{{ Context $v }}:
				t.{{ Context $v }} = stack[i-1]
				t.elementContext = stack[i+1]
			case *slice{{ Context $v }}:
				t.{{ Context $v }} = stack[i-1]
				t.elementContext = stack[i+1]
			case *scalar{{ Context $v }}:
				t.{{ Context $v }} = stack[i-1]
				t.elementContext = stack[i+1]
			case *typeCheck{{ Context $v }}:
				t.{{ Context $v }} = stack[i-1]
			default:
				panic(fmt.Errorf("unsupported type: %+v", t))
		}
	}
	return stack[1]
}

var scalar{{ Context $v }}Pool = sync.Pool{
	New: func() interface{} {return &scalar{{ Context $v }}{}},
}

func newScalar{{ Context $v }}() {{ Context $v }} {
	ret := scalar{{ Context $v }}Pool.Get().(*scalar{{ Context $v }})
	*ret = scalar{{ Context $v }}{}
	return ret
}

var pointer{{ Context $v }}Pool = sync.Pool{
	New: func() interface{} { return &pointer{{ Context $v}}{}},
}

func newPointer{{ Context $v }}(takeAddr bool) {{ Context $v }} {
	ret := pointer{{ Context $v }}Pool.Get().(*pointer{{ Context $v }})
	*ret = pointer{{ Context $v }}{takeAddr: takeAddr}
	return ret
}


var slice{{ Context $v }}Pool = sync.Pool{
	New: func() interface{} {return &slice{{ Context $v }}{}},
}

func newSlice{{ Context $v }}() {{ Context $v }} {
	ret := slice{{ Context $v }}Pool.Get().(*slice{{ Context $v }})
	*ret = slice{{ Context $v }}{}
	return ret
}

var typeCheck{{ Context $v }}Pool = sync.Pool{
	New: func() interface{} {return &typeCheck{{ Context $v }}{}},
}

func newTypeCheck{{ Context $v }}(assignableTo reflect.Type) {{ Context $v }} {
	ret := typeCheck{{ Context $v }}Pool.Get().(*typeCheck{{ Context $v }})
	*ret = typeCheck{{ Context $v }}{assignableTo: assignableTo}
	return ret
}
`
}
