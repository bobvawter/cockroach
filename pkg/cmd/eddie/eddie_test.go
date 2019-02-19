package main

import (
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test(t *testing.T) {
	a := assert.New(t)

	exe, err := ioutil.TempFile("", "eddie")
	if !a.NoError(err) {
		return
	}

	e := Eddie{
		Dir:      "./demo",
		Outfile:  exe.Name(),
		Packages: []string{"."},
	}

	a.NoError(e.Execute())
}
