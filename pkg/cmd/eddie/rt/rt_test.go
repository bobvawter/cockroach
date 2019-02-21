// Copyright 2019 The Cockroach Authors.
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
// permissions and limitations under the License.

package rt

import (
	"context"
	"log"
	"os"
	"testing"

	"github.com/cockroachdb/cockroach/pkg/cmd/eddie/demo"
	"github.com/cockroachdb/cockroach/pkg/cmd/eddie/ext"
	"github.com/stretchr/testify/assert"
)

// This test creates a statically-configured Enforcer using the demo package.
func Test(t *testing.T) {
	a := assert.New(t)

	e := &Enforcer{
		Contracts: map[string]func() ext.Contract{
			"MustReturnInt": func() ext.Contract { return &demo.MustReturnInt{} },
		},
		Dir:      "../demo",
		Logger:   log.New(os.Stdout, "", 0),
		Packages: []string{"."},
		Tests:    true,
	}

	a.NoError(e.execute(context.Background()))

	a.Len(e.aliases, 1)
	a.Len(e.assertions, 4)
	a.Len(e.targets, 7)
}
