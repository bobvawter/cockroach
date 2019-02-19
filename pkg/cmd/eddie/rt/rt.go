package rt

import (
	"fmt"
	"os"

	"github.com/cockroachdb/cockroach/pkg/cmd/eddie/ext"
	"github.com/spf13/cobra"
)

type Enforcer struct {
	// Contracts contains providers for the various Contract types.
	// This map is the primary point of code-generation.
	Contracts map[string]func() ext.Contract
	Name      string
}

func (e *Enforcer) Main() {
	root := cobra.Command{
		Use:          e.Name,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
	}

	root.AddCommand(
		&cobra.Command{
			Use:   "contracts",
			Short: "Lists all defined contracts",
			Run: func(cmd *cobra.Command, _ []string) {
				for name := range e.Contracts {
					cmd.Println(name)
				}
			},
		})

	if err := root.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	os.Exit(0)
}

func (e *Enforcer) execute() {
	// Load the source

	// Look for contract declarations
	// - Need to handle "forward-declared" contract aliases.
	// - Want to build up the datastructures that make the next pass easier

	// Aggregate contract declarations and resulting members.
}
