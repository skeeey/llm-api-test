package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// version and commit describe this build. Release binaries get real values
// stamped in with -ldflags -X; a plain go build keeps the defaults.
var (
	version = "dev"
	commit  = "none"
)

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the version and commit of this build",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Fprintf(cmd.OutOrStdout(), "llm-api-test %s (commit %s)\n", version, commit)
			return nil
		},
	}
}
