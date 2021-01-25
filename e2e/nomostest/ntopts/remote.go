package ntopts

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/nomos/pkg/client/restconfig"
)

// RemoteCluster tells the test to use the remote cluster pointed to by the config flags.
func RemoteCluster(t *testing.T) Opt {
	return func(opt *New) {
		t.Helper()

		restConfig, err := restconfig.NewRestConfig()
		if err != nil {
			t.Fatal(err)
		}
		opt.RESTConfig = restConfig

		kcfgPath := filepath.Join(opt.TmpDir, Kubeconfig)

		kubeconfig := os.Getenv(Kubeconfig)
		if len(kubeconfig) == 0 {
			home, err := os.UserHomeDir()
			if err != nil {
				t.Fatal(err)
			}
			kubeconfig = filepath.Join(home, ".kube", "config")
		}

		data, err := ioutil.ReadFile(kubeconfig)
		if err != nil {
			t.Fatal(err)
		}
		err = ioutil.WriteFile(kcfgPath, data, os.ModePerm)
		if err != nil {
			t.Fatal(err)
		}
	}
}
