package bugreport

import (
	"archive/zip"
	"bufio"
	"context"
	"flag"
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
)

type reporter struct {
	writer        *zip.Writer
	name          string
	errorList     []error
	writingErrors []error
}

// Cmd retrieves readers for all relevant nomos container logs and cluster state commands and writes them to a zip file
var Cmd = &cobra.Command{
	Use:   "bugreport",
	Short: fmt.Sprintf("Generates a zip file of relevant %v debug information.", configmanagement.CLIName),
	Long:  "Generates a zip file in your current directory containing an aggregate of the logs and cluster state for debugging purposes.",
	Run: func(cmd *cobra.Command, args []string) {
		// hack to set the hidden variable in glog to also print info statements
		// cobra does not expose core golang-style flags
		if err := flag.CommandLine.Parse([]string{"--stderrthreshold=0"}); err != nil {
			glog.Errorf("could not increase logging verbosity: %v", err)
		}

		cfg, err := restconfig.NewRestConfig()
		if err != nil {
			glog.Fatalf("failed to create rest config: %v", err)
		}

		br, err := bugreport.New(context.Background(), cfg)
		if err != nil {
			glog.Fatalf("failed to initialize bug reporter: %v", err)
		}

		zipName := getReportName()
		outFile, err := os.Create(zipName)
		if err != nil {
			glog.Fatalf("failed to create file %v: %v", zipName, err)
		}
		report := reporter{
			writer:        zip.NewWriter(outFile),
			name:          zipName,
			errorList:     []error{},
			writingErrors: []error{},
		}

		report.writeRawInZip(br.FetchLogSources())
		report.writeRawInZip(br.FetchCMResources())

		currentk8sContext, err := restconfig.CurrentContextName()
		if err != nil {
			report.errorList = append(report.errorList, err)
		}
		var k8sContexts = []string{currentk8sContext}

		report.addNomosStatusToZip(k8sContexts)
		report.addNomosVersionToZip(k8sContexts)

		err = report.writer.Close()
		if err != nil {
			e := fmt.Errorf("failed to close zip writer: %v", err)
			report.errorList = append(report.errorList, e)
		}

		err = outFile.Close()
		if err != nil {
			e := fmt.Errorf("failed to close zip file: %v", err)
			report.errorList = append(report.errorList, e)
		}

		if len(report.writingErrors) == 0 {
			glog.Infof("Bug report written to zip file: %v\n", zipName)
		} else {
			glog.Warningf("Some errors returned while writing zip file.  May exist at: %v\n", zipName)
		}
		report.errorList = append(report.errorList, report.writingErrors...)

		if len(report.errorList) > 0 {
			for _, e := range report.errorList {
				glog.Errorf("Error: %v\n", e)
			}

			glog.Errorf("Partial bug report may have succeeded.  Look for file: %s\n", zipName)
		} else {
			fmt.Println("Created file " + zipName)
		}
	},
}

func (r *reporter) addNomosStatusToZip(k8sContexts []string) {
	if statusRc, err := status.GetStatusReadCloser(k8sContexts); err != nil {
		r.errorList = append(r.errorList, err)
	} else if err = r.writeReadableToZip(bugreport.Readable{
		Name:       "processed/status",
		ReadCloser: statusRc,
	}); err != nil {
		r.writingErrors = append(r.writingErrors, err)
	}
}

func (r *reporter) addNomosVersionToZip(k8sContexts []string) {
	if versionRc, err := version.GetVersionReadCloser(k8sContexts); err != nil {
		r.errorList = append(r.errorList, err)
	} else if err = r.writeReadableToZip(bugreport.Readable{
		Name:       "processed/version",
		ReadCloser: versionRc,
	}); err != nil {
		r.writingErrors = append(r.writingErrors, err)
	}
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

//TODO join kubecontext for file path
func (r *reporter) writeReadableToZip(readable bugreport.Readable) error {
	baseName := filepath.Base(r.name)
	dirName := strings.TrimSuffix(baseName, filepath.Ext(baseName))
	fileName := filepath.FromSlash(filepath.Join(dirName, readable.Name) + ".txt")
	f, err := r.writer.Create(fileName)
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

	fmt.Println("Wrote file " + fileName)

	return nil
}

func (r *reporter) writeRawInZip(toBeRead []bugreport.Readable, errs []error) {
	if len(errs) > 0 {
		r.errorList = append(r.errorList, errs...)
	}

	for _, readable := range toBeRead {
		err := r.writeReadableToZip(readable)
		if err != nil {
			r.writingErrors = append(r.writingErrors, err)
		}
	}

}
