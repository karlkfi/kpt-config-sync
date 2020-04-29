package bugreport

import (
	"archive/zip"
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/golang/glog"
	"github.com/google/nomos/pkg/api/configmanagement"
	"github.com/google/nomos/pkg/bugreport"
	"github.com/google/nomos/pkg/client/restconfig"
	"github.com/spf13/cobra"
	"k8s.io/client-go/kubernetes"
)

// Cmd retrieves readers for all relevant nomos container logs and writes the logs to a zip file
var Cmd = &cobra.Command{
	Use:   "bugreport",
	Short: fmt.Sprintf("Generates a zip file of relevant %v debug information.", configmanagement.CLIName),
	Long:  "Generates a zip file in your current directory containing an aggregate of the logs and cluster state for debugging purposes.",
	Run: func(cmd *cobra.Command, args []string) {
		cfg, err := restconfig.NewRestConfig()
		if err != nil {
			glog.Fatalf("failed to create rest config: %v", err)
		}

		clientSet, err := kubernetes.NewForConfig(cfg)
		if err != nil {
			glog.Fatalf("failed to create k8s client: %v", err)
		}

		toBeRead, errorList := bugreport.FetchLogSources(clientSet)

		// write from readers into the zip
		zipName := getReportName()
		writingErrors := writeReadablesToZip(toBeRead, zipName)
		if len(writingErrors) == 0 {
			glog.Infof("Bug report written to zip file: %v\n", zipName)
		} else {
			glog.Warningf("Some errors returned while writing zip file.  May exist at: %v\n", zipName)
		}
		errorList = append(errorList, writingErrors...)

		if len(errorList) > 0 {
			for _, e := range errorList {
				glog.Errorf("Error: %v\n", e)
			}

			glog.Fatalf("Partial bug report may have succeeded.  Look for file: %s\n", zipName)
		}
	},
}

func getReportName() string {
	now := time.Now()
	baseName := fmt.Sprintf("bug_report_%v.zip", now.Unix())
	nameWithPath, err := filepath.Abs(baseName)
	if err != nil {
		nameWithPath = baseName
	}

	return nameWithPath
}

func writeReadablesToZip(toBeRead []bugreport.Readable, fileName string) []error {
	var errorList []error

	outFile, err := os.Create(fileName)
	if err != nil {
		e := fmt.Errorf("failed to create file %v: %v", fileName, err)
		errorList = append(errorList, e)
	}

	baseName := filepath.Base(fileName)
	dirName := strings.TrimSuffix(baseName, filepath.Ext(baseName))
	zipWriter := zip.NewWriter(outFile)

	for _, readable := range toBeRead {
		fileName := filepath.Join(dirName, readable.Name) + ".txt"
		f, err := zipWriter.Create(fileName)
		if err != nil {
			e := fmt.Errorf("failed to create file %v inside zip: %v", fileName, err)
			errorList = append(errorList, e)
			continue
		}

		w := bufio.NewWriter(f)
		_, err = w.ReadFrom(readable.ReadCloser)
		if err != nil {
			e := fmt.Errorf("failed to write file %v to zip: %v", fileName, err)
			errorList = append(errorList, e)
			continue
		}

		err = w.Flush()
		if err != nil {
			e := fmt.Errorf("failed to flush writer to zip for file %v:i %v", fileName, err)
			errorList = append(errorList, e)
			continue
		}
	}

	err = zipWriter.Close()
	if err != nil {
		e := fmt.Errorf("failed to close zip writer: %v", err)
		errorList = append(errorList, e)
	}

	err = outFile.Close()
	if err != nil {
		e := fmt.Errorf("failed to close zip file: %v", err)
		errorList = append(errorList, e)
	}

	return errorList
}
