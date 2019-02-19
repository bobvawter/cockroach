package gen

import (
	"io/ioutil"
	"plugin"
	"testing"

	"github.com/cockroachdb/cockroach/pkg/cmd/eddie/rt"
	"github.com/stretchr/testify/assert"
)

// This test will compile an enforcer from the contents of the demo
// directory as a go plugin, then dynamically load it to continue
// testing.
func TestCompileAndLoad(t *testing.T) {
	a := assert.New(t)

	// Set up a temp file to hold the generated file.
	exe, err := ioutil.TempFile("", "eddie")
	if !a.NoError(err) {
		return
	}

	e := Eddie{
		Name:     "gen_test",
		Outfile:  exe.Name(),
		Packages: []string{"../demo"},
		plugin:   true,
	}
	a.NoError(e.Execute())
	a.Len(e.contracts, 1)

	// Now try to open the plugin.
	plg, err := plugin.Open(exe.Name())
	if !a.NoError(err) {
		return
	}
	// Look for the top-level var in the generated code.
	sym, err := plg.Lookup("Enforcer")
	if !a.NoError(err) {
		return
	}

	impl := sym.(*rt.Enforcer)
	a.Len(impl.Contracts, len(e.contracts))
}
