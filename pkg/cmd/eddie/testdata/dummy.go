package testdata

import "github.com/cockroachdb/cockroach/pkg/cmd/eddie/ext"

var _ ext.Contract = MyContract{}
var _ ext.Contract = &MyContract2{}

// This will be ignored, since there's no actual assertion.
var _ = NotAsserted{}

type MyContract struct{}

func (MyContract) Enforce(ctx ext.Context) {}

type MyContract2 struct{}

func (*MyContract2) Enforce(ctx ext.Context) {}

type NotAsserted struct{}

func (NotAsserted) Enforce(ctx ext.Context) {}
