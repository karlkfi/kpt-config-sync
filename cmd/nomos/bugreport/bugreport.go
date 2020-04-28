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
	"github.com/google/nomos/cmd/nomos/status"
	"github.com/google/nomos/cmd/nomos/version"
	"github.com/google/nomos/pkg/api/configmanagement"
	"github.com/google/nomos/pkg/bugreport"
	"github.com/google/nomos/pkg/client/restconfig"
	"github.com/spf13/cobra"
	"k8s.io/client-go/kubernetes"
)

// Cmd retrieves readers for all relevant nomos container logs and cluster state commands and writes them to a zip file
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

		var errorList []error
		var writingErrors []error

		zipName := getReportName()
		outFile, err := os.Create(zipName)
		if err != nil {
			e := fmt.Errorf("failed to create file %v: %v", zipName, err)
			errorList = append(errorList, e)
		}
		zipWriter := zip.NewWriter(outFile)

		toBeRead, errs := bugreport.FetchLogSources(clientSet)
		if len(errs) > 0 {
			errorList = append(errorList, errs...)
		}

		for _, readable := range toBeRead {
			err := writeReadableToZip(readable, zipWriter, zipName)
			if err != nil {
				writingErrors = append(writingErrors, err)
			}
		}

		currentk8sContext, err := restconfig.CurrentContextName()
		if err != nil {
			errorList = append(errorList, err)
		}
		var k8sContexts = []string{currentk8sContext}

		err = addNomosStatusToZip(k8sContexts, zipWriter, zipName)
		if err != nil {
			writingErrors = append(writingErrors, err)
		}

		err = addNomosVersionToZip(k8sContexts, zipWriter, zipName)
		if err != nil {
			writingErrors = append(writingErrors, err)
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

func addNomosStatusToZip(k8sContexts []string, zipWriter *zip.Writer, zipName string) error {
	statusRc, err := status.GetStatusReadCloser(k8sContexts)
	if err != nil {
		return err
	}

	return writeReadableToZip(bugreport.Readable{
		Name:       "processed/status",
		ReadCloser: statusRc,
	}, zipWriter, zipName)
}

func addNomosVersionToZip(k8sContexts []string, zipWriter *zip.Writer, zipName string) error {
	versionRc, err := version.GetVersionReadCloser(k8sContexts)
	if err != nil {
		return err
	}

	return writeReadableToZip(bugreport.Readable{
		Name:       "processed/version",
		ReadCloser: versionRc,
	}, zipWriter, zipName)
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

func writeReadableToZip(readable bugreport.Readable, zipWriter *zip.Writer, zipName string) error {
	baseName := filepath.Base(zipName)
	dirName := strings.TrimSuffix(baseName, filepath.Ext(baseName))
	fileName := filepath.FromSlash(filepath.Join(dirName, readable.Name) + ".txt")
	f, err := zipWriter.Create(fileName)
	if err != nil {
		e := fmt.Errorf("failed to create file %v inside zip: %v", fileName, err)
		return e
	}

	w := bufio.NewWriter(f)
	_, err = w.ReadFrom(readable.ReadCloser)
	if err != nil {
		e := fmt.Errorf("failed to write file %v to zip: %v", fileName, err)
		return e
	}

	err = w.Flush()
	if err != nil {
		e := fmt.Errorf("failed to flush writer to zip for file %v:i %v", fileName, err)
		return e
	}

	return nil
}
