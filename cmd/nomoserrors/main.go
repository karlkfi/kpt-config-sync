package main

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/google/nomos/cmd/nomoserrors/examples"
	"github.com/google/nomos/pkg/status"
	"github.com/spf13/cobra"
)

var idFlag string

var rootCmd = &cobra.Command{
	Use:   "nomoserrors",
	Short: "List all error codes and example errors",
	Args:  cobra.ExactArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		examples := examples.Generate()

		if idFlag == "" {
			printErrorCodes()
		}
		idFlag = strings.TrimPrefix(idFlag, "KNV")
		printErrors(idFlag, examples)
	},
}

func init() {
	rootCmd.Flags().StringVar(&idFlag, "id", "", "if set, only print errors for the passed ID")
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

func printErrorCodes() {
	fmt.Println("=== USED ERROR CODES ===")
	for _, code := range status.CodeRegistry() {
		fmt.Println(code)
	}
	fmt.Println()
}

func printErrors(id string, e map[string][]status.Error) {
	fmt.Println("=== SAMPLE ERRORS ===")
	fmt.Println()
	for _, err := range sortedErrors(e) {
		if id == "" || err.Code() == id {
			fmt.Println(err.Error())
			fmt.Println()
			fmt.Println()
		}
	}
}
