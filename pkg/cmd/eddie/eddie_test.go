package main

import "testing"

func Test(t *testing.T) {
	e := Eddie{
		Dir:      "./testdata",
		Packages: []string{"."},
	}
	if err := e.Execute(); err != nil {
		t.Fatal(err)
	}
}
