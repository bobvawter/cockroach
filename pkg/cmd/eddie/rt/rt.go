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
