package main

import (
	"fmt"
	"os"
	"sort"

	_ "github.com/google/nomos/pkg/importer/analyzer/vet" // required for vet errors
	_ "github.com/google/nomos/pkg/importer/id"           // required for id errors
	"github.com/google/nomos/pkg/status"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "nomoserrors",
	Short: "List all error codes and examples",
	Args:  cobra.ExactArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		printErrorCodes()
		printErrors()
	},
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

type errEntry struct {
	code string
	err  status.Error
}

func sortedErrors() []errEntry {
	var errs []errEntry
	for code, err := range status.Registry() {
		errs = append(errs, errEntry{code: code, err: err})
	}
	sort.Slice(errs, func(i, j int) bool {
		return errs[i].code < errs[j].code
	})
	return errs
}

func printErrorCodes() {
	fmt.Println("=== ERROR CODES ===")
	for _, err := range sortedErrors() {
		if err.err == nil {
			fmt.Printf("%s: RESERVED\n", err.code)
		} else {
			fmt.Printf("%s: %T\n", err.code, err.err)
		}
	}
	fmt.Println()
}

func printErrors() {
	fmt.Println("=== SAMPLE ERRORS ===")
	for _, err := range sortedErrors() {
		if err.err != nil {
			fmt.Println(err.err.Error())
			fmt.Println()
		}
	}
}
