package version

import (
	"fmt"

	"github.com/google/nomos/pkg/version"
	"github.com/spf13/cobra"
)

// Cmd is the Cobra object representing the nomos version command.
var Cmd = &cobra.Command{
	Use:   "version",
	Short: "Prints the version of this binary",
	Long: `Prints the version of the "nomos" client binary for debugging purposes.
`,
	Example: `  nomos version`,
	Run: func(_ *cobra.Command, _ []string) {
		fmt.Printf("%s\n", version.VERSION)
	},
}
