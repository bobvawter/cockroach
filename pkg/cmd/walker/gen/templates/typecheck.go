package templates

func init() {
	TemplateSources["typecheck"] = `
{{- $v := . -}}
// ------------------- Type Checking Context ---------------------------

// typeCheck{{ Context $v }} is the context facade exposed to user-code.
// It is also responsible for pushing and popping {{ Location $v }}.
type typeCheck{{ Context $v }} struct {
	{{ Context $v }}
	assignableTo reflect.Type
}

var _ {{ Context $v }} = &typeCheck{{ Context $v }}{}


func (c *typeCheck{{ Context $v }}) accept(
	v {{ Visitor $v }}, val interface{},
) (result interface{}, changed bool) {
	c.rawStack().Push({{ Location $v }}{
		Value: val.({{ Intf $v }}),
		Index: -1,
	})

	x := val.({{ Impl $v }})
	recurse, err := x.pre(c, v)
	if err != nil {
		c.abort(err)
	}
	if recurse {
		x.traverse(c, v)
	}
	if err := x.post(c, v); err != nil {
		c.abort(err)
	}
	c.rawStack().Pop()
	return x, false
}

// check verifies that val is assignable to the configured type.
func (c *typeCheck{{ Context $v }}) check(val {{ Intf $v }}) {
	valTyp := reflect.TypeOf(val)
	if !valTyp.AssignableTo(c.assignableTo) {
		c.abort(fmt.Errorf("%s is not assignable to %s", valTyp, c.assignableTo))
	}
}

func (c *typeCheck{{ Context $v }}) close() {
	*c = typeCheck{{ Context $v }}{}
	typeCheck{{ Context $v }}Pool.Put(c)
}

func (c *typeCheck{{ Context $v }}) InsertAfter(n {{ Intf $v }}) {
	c.check(n)
	c.{{ Context $v }}.InsertAfter(n)
}

func (c *typeCheck{{ Context $v }}) InsertBefore(n {{ Intf $v }}) {
	c.check(n)
	c.{{ Context $v }}.InsertBefore(n)
}

func (c *typeCheck{{ Context $v }}) Replace(n {{ Intf $v }}) {
	c.check(n)
	c.{{ Context $v }}.Replace(n)
}
`
}
