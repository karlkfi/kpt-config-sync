package util

import (
	"fmt"
	"io"
	"strings"
)

const (
	// Indent is the extra spaces for indentation.
	Indent = "  "
	// Separator is the delimiter before each cluster.
	Separator = "--------------------"
	// Bullet is the separator before each bullet item.
	Bullet = "- "

	// ColorDefault is the default color code
	ColorDefault = "\033[0m"
	// ColorRed is the red color code
	ColorRed = "\033[31m"
	// ColorGreen is the green color code
	ColorGreen = "\033[32m"
	// ColorYellow is the yellow color code
	ColorYellow = "\033[33m"
	// ColorCyan is the cyan color code
	ColorCyan = "\033[36m"
)

// MonoRepoNotice logs a notice for the clusters that are running in the legacy mode.
func MonoRepoNotice(writer io.Writer, monoRepoClusters ...string) {
	clusterCount := len(monoRepoClusters)
	if clusterCount != 0 {
		if clusterCount == 1 {
			fmt.Fprintf(writer, "%sNotice: The cluster %q is still running in the legacy mode.\n",
				ColorYellow, monoRepoClusters[0])
		} else {
			fmt.Fprintf(writer, "%sNotice: The following clusters are still running in the legacy mode:\n%s%s\n",
				ColorYellow, Bullet, strings.Join(monoRepoClusters, "\n"+Bullet))
		}
		fmt.Fprintf(writer, "Run `nomos migrate` to enable multi-repo mode. It provides you with additional features and gives you the flexibility to sync to a single repository, or multiple repositories.%s\n", ColorDefault)
	}
}
