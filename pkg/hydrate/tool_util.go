package hydrate

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/Masterminds/semver"
	"github.com/pkg/errors"
)

const (
	// HelmVersion is the minimum required version of Helm for hydration.
	HelmVersion = "v3.5.3"
	// KustomizeVersion is the minimum required version of Kustomize for hydration.
	KustomizeVersion = "v4.1.3"
	// Helm is the binary name of the installed Helm.
	Helm = "helm"
	// Kustomize is the binary name of the installed Kustomize.
	Kustomize = "kustomize"
)

var (
	semverRegex             = regexp.MustCompile(semver.SemVerRegex)
	validKustomizationFiles = []string{"kustomization.yaml", "kustomization.yml", "Kustomization"}
)

// NeedsKustomize checks if there is a Kustomization config file under the directory.
func NeedsKustomize(dir string) (bool, error) {
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		return false, errors.Wrapf(err, "unable to traverse the directory: %s", dir)
	}
	for _, f := range files {
		if hasKustomization(filepath.Base(f.Name())) {
			return true, nil
		}
	}
	return false, nil
}

// hasKustomization checks if the file is a Kustomize configuration file.
func hasKustomization(filename string) bool {
	for _, kustomization := range validKustomizationFiles {
		if filename == kustomization {
			return true
		}
	}
	return false
}

// KustomizeBuild runs the 'kustomize build' command to render the configs.
func KustomizeBuild(input, output string) error {
	// The `--enable-alpha-plugins` and `--enable-exec` flags are to support rendering
	// Helm charts using the Helm inflation function, see go/kust-helm-for-config-sync.
	// The `--enable-helm` flag is to enable use of the Helm chart inflator generator.
	// We decided to enable all the flags so that both the Helm plugin and Helm
	// inflation function are supported. This provides us with a fallback plan
	// if the new Helm inflation function is having issues.
	// It has no side-effect if no Helm chart in the DRY configs.
	args := []string{"build", input, "--enable-alpha-plugins", "--enable-exec", "--enable-helm", "--output", output}

	if _, err := os.Stat(output); err == nil {
		if err := os.RemoveAll(output); err != nil {
			return errors.Wrapf(err, "unable to delete directory: %s", output)
		}
	}

	fileMode := os.FileMode(0755)
	if err := os.MkdirAll(output, fileMode); err != nil {
		return errors.Wrapf(err, "unable to make directory: %s", output)
	}

	out, err := runCommand("", Kustomize, args...)
	if err != nil {
		if err := os.RemoveAll(output); err != nil {
			return errors.Wrapf(err, "unable to delete directory: %s", output)
		}
		return errors.Wrapf(err, "failed to run kustomize build, stdout: %s", out)
	}
	return nil
}

func runCommand(cwd, command string, args ...string) (string, error) {
	cmdStr := command + " " + strings.Join(args, " ")

	cmd := exec.Command(command, args...)
	if cwd != "" {
		cmd.Dir = cwd
	}
	outbuf := bytes.NewBuffer(nil)
	errbuf := bytes.NewBuffer(nil)
	cmd.Stdout = outbuf
	cmd.Stderr = errbuf

	err := cmd.Run()
	stdout := outbuf.String()
	stderr := errbuf.String()
	if err != nil {
		return "", fmt.Errorf("run(%s): %w: { stdout: %q, stderr: %q }", cmdStr, err, stdout, stderr)
	}

	return stdout, nil
}

// ValidateTool checks if the hydration tool is installed and if the installed
// version meets the required version.
func ValidateTool(tool, requiredVersion string) error {
	detectedVersion, err := getSemver(tool)
	if err != nil {
		return err
	}
	requiredSemVersion, err := semver.NewVersion(requiredVersion)
	if err != nil {
		return err
	}
	if detectedVersion.LessThan(requiredSemVersion) {
		return errors.Errorf("the current %s version is %q. Please upgrade to version %s+", tool, detectedVersion, requiredVersion)
	}
	return nil
}

var toolVersion = getVersion

func getSemver(tool string) (*semver.Version, error) {
	version, err := toolVersion(tool)
	if err != nil {
		return nil, err
	}

	matches := semverRegex.FindStringSubmatch(version)
	if len(matches) == 0 {
		return nil, fmt.Errorf("unable to extract semver from %q", version)
	}
	return semver.NewVersion(matches[0])
}

func getVersion(tool string) (string, error) {
	args := []string{"version", "--short"}
	out, err := exec.Command(tool, args...).CombinedOutput()
	if err != nil {
		return "", errors.Wrapf(err, "unable to get %s version", tool)
	}
	version := strings.TrimSpace(string(out))
	// remove the curly braces for the kustomize output
	version = strings.TrimPrefix(version, "{")
	version = strings.TrimSuffix(version, "}")
	version = strings.TrimSpace(version)
	// remove the leading 'kustomize/' prefix for the kustomize output
	version = strings.TrimPrefix(version, "kustomize/")
	return version, nil
}
