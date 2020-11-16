package main

import (
	"flag"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

const delimiter = "---\n"

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
		if !strings.HasSuffix(string(readBytes), "\n") {
			readBytes = append(readBytes, []byte("\n")...)
		}

		yamlResources = append(yamlResources, string(readBytes))
	}

	// overwrite the file with our new contents
	byteOut := []byte(strings.Join(yamlResources, delimiter))
	err := ioutil.WriteFile(*destination, byteOut, 0644)
	if err != nil {
		panic(err)
	}
}
