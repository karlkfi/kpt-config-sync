package ntopts

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/nomos/e2e"
	"github.com/google/nomos/pkg/importer"
)

// RemoteCluster tells the test to use the remote cluster pointed to by the
// default context instead of creating a Kind cluster.
func RemoteCluster(t *testing.T) Opt {
	if !*e2e.Manual {
		t.Skip("Must pass --manual so this isn't accidentally run against a test cluster automatically.")
	}

	return func(opt *New) {
		t.Helper()

		restConfig, err := importer.DefaultCLIOptions.ToRESTConfig()
		if err != nil {
			t.Fatal(err)
		}
		opt.RESTConfig = restConfig

		kcfgPath := filepath.Join(opt.TmpDir, Kubeconfig)
		home, err := os.UserHomeDir()
		if err != nil {
			t.Fatal(err)
		}
		data, err := ioutil.ReadFile(filepath.Join(home, ".kube", "config"))
		if err != nil {
			t.Fatal(err)
		}
		err = ioutil.WriteFile(kcfgPath, data, os.ModePerm)
		if err != nil {
			t.Fatal(err)
		}
	}
}
