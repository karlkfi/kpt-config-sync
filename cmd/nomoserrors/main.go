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

func sortedErrors() []status.Error {
	var allErrs []status.Error
	for _, errs := range status.Registry() {
		allErrs = append(allErrs, errs...)
	}
	sort.Slice(allErrs, func(i, j int) bool {
		return allErrs[i].Error() < allErrs[j].Error()
	})
	return allErrs
}

func printErrorCodes() {
	fmt.Println("=== USED ERROR CODES ===")
	for code := range status.Registry() {
		// TODO: Need to rethink how to document error names.
		fmt.Printf("%s", code)
	}
	fmt.Println()
}

func printErrors() {
	fmt.Println("=== SAMPLE ERRORS ===")
	for _, err := range sortedErrors() {
		fmt.Println(err.Error())
		fmt.Println()
	}
}
