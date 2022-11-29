package cmd

import (
	"github.com/spf13/cobra"

	"go.k6.io/k6/cmd/state"
	"go.k6.io/k6/lib/consts"
)

func getCmdVersion(gs *state.GlobalState) *cobra.Command {
	// versionCmd represents the version command.
	return &cobra.Command{
		Use:   "version",
		Short: "Show application version",
		Long:  `Show the application version and exit.`,
		Run: func(_ *cobra.Command, _ []string) {
			gs.Console.Printf("k6 v%s\n", consts.FullVersion())
		},
	}
}
