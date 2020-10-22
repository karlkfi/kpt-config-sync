package vet

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/google/nomos/cmd/nomos/flags"
	"github.com/google/nomos/pkg/importer/filesystem"
	"github.com/google/nomos/pkg/importer/filesystem/cmpath"
	ft "github.com/google/nomos/pkg/importer/filesystem/filesystemtest"
)

func resetFlags() {
	// Flags are global state carried over between tests.
	// Cobra lazily evaluates flags only if they are declared, so unless these
	// are reset, successive calls to Cmd.Execute aren't guaranteed to be
	// independent.
	flags.Clusters = nil
	flags.Path = flags.PathDefault
	flags.SkipAPIServer = false

	sourceFormatValue = string(filesystem.SourceFormatHierarchy)
	namespaceValue = ""
}

var examplesDir = cmpath.RelativeSlash("../../../examples")

func TestVet_Acme(t *testing.T) {
	resetFlags()
	Cmd.SilenceUsage = true

	os.Args = []string{
		"vet", // this first argument does nothing, but is required to exist.
		"--path", examplesDir.Join(cmpath.RelativeSlash("acme")).OSPath(),
	}

	err := Cmd.Execute()
	if err != nil {
		t.Error(err)
	}
}

func TestVet_AcmeSymlink(t *testing.T) {
	resetFlags()
	Cmd.SilenceUsage = true

	dir := ft.NewTestDir(t)
	symDir := dir.Root().Join(cmpath.RelativeSlash("acme-symlink"))

	absExamples, err := filepath.Abs(examplesDir.Join(cmpath.RelativeSlash("acme")).OSPath())
	if err != nil {
		t.Fatal(err)
	}
	err = os.Symlink(absExamples, symDir.OSPath())
	if err != nil {
		t.Fatal(err)
	}

	os.Args = []string{
		"vet", // this first argument does nothing, but is required to exist.
		"--path", symDir.OSPath(),
	}

	err = Cmd.Execute()
	if err != nil {
		t.Error(err)
	}
}

func TestVet_FooCorp(t *testing.T) {
	resetFlags()
	Cmd.SilenceUsage = true

	os.Args = []string{
		"vet", // this first argument does nothing, but is required to exist.
		"--path", examplesDir.Join(cmpath.RelativeSlash("foo-corp-example/foo-corp")).OSPath(),
	}

	err := Cmd.Execute()
	if err != nil {
		t.Error(err)
	}
}

func TestVet_MultiCluster(t *testing.T) {
	Cmd.SilenceUsage = true

	tcs := []struct {
		name      string
		args      []string
		wantError bool
	}{
		{
			name:      "detect collision when all clusters enabled",
			wantError: true,
		},
		{
			name:      "detect collision in prod-cluster",
			args:      []string{"--clusters", "prod-cluster"},
			wantError: true,
		},
		{
			name:      "do not detect collision in dev-cluster",
			args:      []string{"--clusters", "dev-cluster"},
			wantError: false,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			resetFlags()

			os.Args = append([]string{
				"vet", // this first argument does nothing, but is required to exist.
				"--path", examplesDir.Join(cmpath.RelativeSlash("parse-errors/cluster-specific-collision")).OSPath(),
			}, tc.args...)

			err := Cmd.Execute()
			if !tc.wantError && err != nil {
				t.Errorf("got vet errors, want nil:\n%v", err)
			} else if tc.wantError && err == nil {
				t.Error("go no vet error, want err")
			}
		})
	}

}
