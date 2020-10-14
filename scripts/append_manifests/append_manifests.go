package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

const delimiter = "---\n"
const label = "# ----- %v -----\n"

var destination = flag.String("destination", "", "Path to the destination file")

func filePaths(path string) ([]string, error) {
	var paths []string

	err := filepath.Walk(path, func(fPath string, info os.FileInfo, err error) error {
		// recursion is not supported
		if info.IsDir() {
			return nil
		}
		paths = append(paths, fPath)
		return nil
	})
	if err != nil {
		return nil, err
	}

	return paths, nil
}

func main() {
	flag.Parse()
	if *destination == "" {
		panic("Must specify a desination path")
	}

	// Ingest input files/dirs as positional arguments
	var paths []string
	for _, inputPath := range flag.Args() {
		fps, err := filePaths(inputPath)
		if err != nil {
			panic(err)
		}
		paths = append(paths, fps...)
	}

	// Append each of the paths to the yamlResources file
	var yamlResources []string
	for _, p := range paths {
		readBytes, err := ioutil.ReadFile(p)
		if err != nil {
			panic(err)
		}

		// If a file doesn't end in a newline, add one
		lastRune, _ := utf8.DecodeLastRuneInString(string(readBytes))
		if lastRune != '\n' {
			readBytes = append(readBytes, []byte("\n")...)
		}

		labeledRes := fmt.Sprintf(label, p) + string(readBytes)
		yamlResources = append(yamlResources, labeledRes)
	}

	// overwrite the file with our new contents
	byteOut := []byte(strings.Join(yamlResources, delimiter))
	err := ioutil.WriteFile(*destination, byteOut, 0644)
	if err != nil {
		panic(err)
	}
}
