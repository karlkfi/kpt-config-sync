package hydrate

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/Masterminds/semver"
	"github.com/golang/glog"
	"github.com/google/nomos/cmd/nomos/flags"
	nomosparse "github.com/google/nomos/cmd/nomos/parse"
	"github.com/google/nomos/pkg/declared"
	"github.com/google/nomos/pkg/importer"
	"github.com/google/nomos/pkg/importer/filesystem"
	"github.com/google/nomos/pkg/importer/filesystem/cmpath"
	"github.com/google/nomos/pkg/reconcilermanager"
	"github.com/google/nomos/pkg/util/discovery"
	"github.com/google/nomos/pkg/validate"
	"github.com/google/nomos/pkg/vet"
	"github.com/pkg/errors"
)

const (
	// HelmVersion is the minimum required version of Helm for hydration.
	HelmVersion = "v3.6.3"
	// KustomizeVersion is the minimum required version of Kustomize for hydration.
	KustomizeVersion = "v4.3.0"
	// Helm is the binary name of the installed Helm.
	Helm = "helm"
	// Kustomize is the binary name of the installed Kustomize.
	Kustomize = "kustomize"

	maxRetries = 5
)

var (
	semverRegex             = regexp.MustCompile(semver.SemVerRegex)
	validKustomizationFiles = []string{"kustomization.yaml", "kustomization.yml", "Kustomization"}
)

// needsKustomize checks if there is a Kustomization config file under the directory.
func needsKustomize(dir string) (bool, error) {
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

// mustDeleteOutput deletes the hydrated output directory with retries.
// It will exit if all attempts failed.
func mustDeleteOutput(err error, output string) {
	retries := 0
	for retries < maxRetries {
		err := os.RemoveAll(output)
		if err == nil {
			return
		}
		glog.Errorf("Unable to delete directory %s: %v", output, err)
		retries++
	}
	if err != nil {
		glog.Error(err)
	}
	glog.Fatalf("Attempted to delete the output directory %s for %d times, but all failed. Exiting now...", output, retries)
}

// kustomizeBuild runs the 'kustomize build' command to render the configs.
func kustomizeBuild(input, output string) error {
	// The `--enable-alpha-plugins` and `--enable-exec` flags are to support rendering
	// Helm charts using the Helm inflation function, see go/kust-helm-for-config-sync.
	// The `--enable-helm` flag is to enable use of the Helm chart inflator generator.
	// We decided to enable all the flags so that both the Helm plugin and Helm
	// inflation function are supported. This provides us with a fallback plan
	// if the new Helm inflation function is having issues.
	// It has no side-effect if no Helm chart in the DRY configs.
	args := []string{"build", input, "--enable-alpha-plugins", "--enable-exec", "--enable-helm", "--output", output}

	if _, err := os.Stat(output); err == nil {
		mustDeleteOutput(err, output)
	}

	fileMode := os.FileMode(0755)
	if err := os.MkdirAll(output, fileMode); err != nil {
		return errors.Wrapf(err, "unable to make directory: %s", output)
	}

	out, err := runCommand("", Kustomize, args...)
	if err != nil {
		kustomizeErr := errors.Wrapf(err, "failed to run kustomize build, stdout: %s", out)
		mustDeleteOutput(kustomizeErr, output)
		return kustomizeErr
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

// validateTool checks if the hydration tool is installed and if the installed
// version meets the required version.
func validateTool(tool, version, requiredVersion string) error {
	matches := semverRegex.FindStringSubmatch(version)
	if len(matches) == 0 {
		return fmt.Errorf("unable to detect %s version from %q. The recommneded version is %s",
			tool, version, requiredVersion)
	}
	detectedVersion, err := semver.NewVersion(matches[0])
	if err != nil {
		return err
	}
	requiredSemVersion, err := semver.NewVersion(requiredVersion)
	if err != nil {
		return err
	}
	if detectedVersion.LessThan(requiredSemVersion) {
		return errors.Errorf("The current %s version is %q. The recommended version is %s. Please upgrade to the %s+ for compatibility.",
			tool, detectedVersion, requiredVersion, requiredVersion)
	}
	return nil
}

func getVersion(tool string) (string, error) {
	args := []string{"version", "--short"}
	out, err := exec.Command(tool, args...).CombinedOutput()
	if err != nil {
		return "", err
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

func validateKustomize() error {
	version, err := getVersion(Kustomize)
	if err != nil {
		return errors.Errorf("Kustomization file is detected, but Kustomize is not installed: %v. Please install Kustomize and re-run the command.", err)
	}
	if err := validateTool(Kustomize, version, KustomizeVersion); err != nil {
		fmt.Printf("WARNING: %v\n", err)
	}
	return nil
}

func validateHelm() error {
	version, err := getVersion(Helm)
	if err != nil {
		// return nil because Helm binary is optional
		// 'kustomize build' will fail if Helm is needed but not installed
		return nil
	}
	if err := validateTool(Helm, version, HelmVersion); err != nil {
		fmt.Printf("WARNING: %v\n", err)
	}
	return nil
}

// ValidateAndRunKustomize validates if the Kustomize and Helm binaries are supported.
// If supported, it copies the source configs to a temp directory, run 'kustomize build',
// save the output to another temp directory, and return the output path for further
// parsing and validation.
func ValidateAndRunKustomize(sourcePath string) (cmpath.Absolute, error) {
	var output = cmpath.Absolute{}
	if err := validateKustomize(); err != nil {
		return output, err
	}
	if err := validateHelm(); err != nil {
		return output, err
	}

	// Copy the source configs to a temp directory in case 'kustomize build' pulls
	// remote configs (e.g. Helm charts) to the source directory.
	tmpDir, err := ioutil.TempDir(os.TempDir(), "source-")
	if err != nil {
		return output, err
	}
	defer func() {
		_ = os.RemoveAll(tmpDir)
	}()
	if err := copyDir(sourcePath, tmpDir); err != nil {
		return output, err
	}

	// Save the 'kustomize build' output to another temp directory for further
	// parsing or validation.
	tmpSrcDir := filepath.Join(tmpDir, filepath.Base(sourcePath))
	tmpHydratedDir, err := ioutil.TempDir(os.TempDir(), "hydrated-")
	if err != nil {
		return output, err
	}

	if err := kustomizeBuild(tmpSrcDir, tmpHydratedDir); err != nil {
		return output, errors.Wrapf(err, "unable to render the source configs in %s", sourcePath)
	}

	return cmpath.AbsoluteOS(tmpHydratedDir)
}

func copyDir(source, dest string) error {
	cmd := exec.Command("cp", "-RL", source, dest)
	_, err := cmd.CombinedOutput()
	if err != nil {
		return errors.Wrapf(err, "unable to copy from source directory %q to destination directory %q ", source, dest)
	}
	return nil
}

// ValidateHydrateFlags validates the hydrate and vet flags.
// It returns the absolute path of the source directory, if hydration is needed, and errors.
func ValidateHydrateFlags(sourceFormat filesystem.SourceFormat) (cmpath.Absolute, bool, error) {
	abs, err := filepath.Abs(flags.Path)
	if err != nil {
		return cmpath.Absolute{}, false, err
	}
	rootDir, err := cmpath.AbsoluteOS(abs)
	if err != nil {
		return cmpath.Absolute{}, false, err
	}
	rootDir, err = rootDir.EvalSymlinks()
	if err != nil {
		return cmpath.Absolute{}, false, err
	}

	switch flags.OutputFormat {
	case flags.OutputYAML, flags.OutputJSON: // do nothing
	default:
		return cmpath.Absolute{}, false, fmt.Errorf("format argument must be %q or %q", flags.OutputYAML, flags.OutputJSON)
	}

	needsKustomize, err := needsKustomize(abs)
	if err != nil {
		return cmpath.Absolute{}, false, errors.Wrapf(err, "unable to check if Kustomize is needed for the source directory: %s", abs)
	}

	if needsKustomize && sourceFormat == filesystem.SourceFormatHierarchy {
		return cmpath.Absolute{}, false, fmt.Errorf("%s must be %s when Kustomization is needed", reconcilermanager.SourceFormat, filesystem.SourceFormatUnstructured)
	}

	return rootDir, needsKustomize, nil
}

// ValidateOptions returns the validate options for nomos hydrate and vet commands.
func ValidateOptions(ctx context.Context, rootDir cmpath.Absolute) (validate.Options, error) {
	var options = validate.Options{}
	syncedCRDs, err := nomosparse.GetSyncedCRDs(ctx, flags.SkipAPIServer)
	if err != nil {
		return options, err
	}

	var serverResourcer discovery.ServerResourcer = discovery.NoOpServerResourcer{}
	var converter *declared.ValueConverter
	if !flags.SkipAPIServer {
		dc, err := importer.DefaultCLIOptions.ToDiscoveryClient()
		if err != nil {
			return options, err
		}
		serverResourcer = dc

		converter, err = declared.NewValueConverter(dc)
		if err != nil {
			return options, err
		}
	}

	addFunc := vet.AddCachedAPIResources(rootDir.Join(vet.APIResourcesPath))

	options.PolicyDir = cmpath.RelativeOS(rootDir.OSPath())
	options.PreviousCRDs = syncedCRDs
	options.BuildScoper = discovery.ScoperBuilder(serverResourcer, addFunc)
	options.Converter = converter
	options.AllowUnknownKinds = flags.SkipAPIServer
	return options, nil
}
