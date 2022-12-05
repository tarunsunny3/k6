package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"go.k6.io/k6/js/modules"
	"go.k6.io/k6/lib/consts"
)

func getCmdVersion(globalState *globalState) *cobra.Command {
	// versionCmd represents the version command.
	return &cobra.Command{
		Use:   "version",
		Short: "Show application version",
		Long:  `Show the application version and exit.`,
		Run: func(_ *cobra.Command, _ []string) {
			printToStdout(globalState, fmt.Sprintf("k6 v%s\n", consts.FullVersion()))
			for path, mod := range modules.GetJSModules() {
				printToStdout(globalState, fmt.Sprintf("extension %s %s\n", path, mod.Version))
			}
		},
	}
}
