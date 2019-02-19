package rt

import (
	"os"

	"github.com/cockroachdb/cockroach/pkg/cmd/eddie/ext"
)

type Enforcer struct {
	// Contracts contains providers for the various Contract types.
	// This map is the primary point of code-generation.
	Contracts map[string]func() ext.Contract
}

func (e *Enforcer) Main() {
	println(len(e.Contracts))

	os.Exit(0)
}

func (e *Enforcer) execute() {
	// Load the source

	// Look for contract declarations
	// - Need to handle "forward-declared" contract aliases.
	// - Want to build up the datastructures that make the next pass easier

	// Aggregate contract declarations and resulting members.
}
