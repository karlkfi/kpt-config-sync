package main

import (
	"fmt"
	"os"
	"sort"

	"github.com/google/nomos/cmd/nomoserrors/examples"
	"github.com/google/nomos/pkg/status"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "nomoserrors",
	Short: "List all error codes and exampleErrors",
	Args:  cobra.ExactArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		examples := examples.Generate()

		printErrorCodes(examples)
		printErrors(examples)
	},
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func sortedErrors(e map[string][]status.Error) []status.Error {
	var allErrs []status.Error
	for _, errs := range e {
		allErrs = append(allErrs, errs...)
	}
	sort.Slice(allErrs, func(i, j int) bool {
		return allErrs[i].Error() < allErrs[j].Error()
	})
	return allErrs
}

func printErrorCodes(e map[string][]status.Error) {
	fmt.Println("=== USED ERROR CODES ===")
	for _, code := range status.CodeRegistry() {
		fmt.Println(code)
	}
	fmt.Println()
}

func printErrors(e map[string][]status.Error) {
	fmt.Println("=== SAMPLE ERRORS ===")
	for _, err := range sortedErrors(e) {
		fmt.Println(err.Error())
		fmt.Println()
	}
}
