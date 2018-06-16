// Package config contains the shared configuration for the installer script.
package config

import (
	"io"
	"io/ioutil"
	"os"
	"strings"

	// TODO(filmil): Revisit once https://github.com/ghodss/yaml/pull/27
	// is resolved.
	"github.com/filmil/yaml"
	"github.com/pkg/errors"
)

const (
	defaultSyncWaitTimeoutSeconds = 15
	defaultSyncBranch             = "master"
	// homeOnHostEnv is the name of the environment variable defined in
	// entrypoint.sh.template that contains the env variable storing the name
	// of the $HOME directory of the user, on the host machine, not the
	// container.
	homeOnHostEnv = "HOME_ON_HOST"
)

// GitConfig contains the configs needed by GitPolicyImporter.
type GitConfig struct {
	// SyncRepo is the git repository name to sync from.
	SyncRepo string `json:"GIT_SYNC_REPO"`

	// UseSSH is true if we are syncing the repo using ssh.
	UseSSH bool `json:"GIT_SYNC_SSH"`

	// PrivateKeyFilename is the filename containing the private key used
	// for SSH authentication.  This entry is communicated through a file
	// to avoid exposing the contents of the secret in the process table.
	// If this entry is empty, SSH will be set to false.
	//
	// This filename may contain a string $HOME, which is expanded by
	// ExpandVarsCopy().
	PrivateKeyFilename string `json:"PRIVATE_KEY_FILENAME,omitempty"`

	// KnownHostsFilename is the filename containing the known hosts SSH
	// file that Kubernetes will use.  Not copied if not defined.
	//
	// This filename may contain a string $HOME, which is expanded by
	// ExpandVarsCopy().
	KnownHostsFilename string `json:"KNOWN_HOSTS_FILENAME,omitempty"`

	// SyncBranch is the branch to sync from.  Default: "master".
	SyncBranch string `json:"GIT_SYNC_BRANCH"`

	// RootPolicyDir is the absolute path of the directory that contains
	// the local policy.  Default: the root directory of the repo.
	RootPolicyDir string `json:"POLICY_DIR"`

	// SyncWaitSeconds is the time duration in seconds between consecutive
	// syncs.  Default: 15 seconds.
	SyncWaitSeconds int64 `json:"GIT_SYNC_WAIT"`

	// CookieFilename is the filename containing the git http cookies file.
	// If this entry is empty, git-sync's GIT_COOKIE_FILE param will be set to false.
	CookieFilename string `json:"GIT_COOKIE_FILENAME,omitempty"`
}

// Empty returns true if the config does not have necessary fields set and
// should be treated as empty.
func (g *GitConfig) Empty() bool {
	return g.SyncRepo == "" && g.RootPolicyDir == ""
}

// GCPConfig contains the configs needed by GCPPolicyImporter.
type GCPConfig struct {
	// OrgID is the GCP organization id. See:
	// https://cloud.google.com/resource-manager/docs/creating-managing-organization#retrieving_your_organization_id
	OrgID string `json:"ORG_ID"`

	// PrivateKeyFilename is the filename containing the GCP service account
	// private key used for accessing GCP Kubernetes Policy API.
	//
	// This filename may contain a string $HOME, which is expanded by
	// ExpandVarsCopy().
	PrivateKeyFilename string `json:"PRIVATE_KEY_FILENAME"`

	// host:port endpoint of the Kubernetes Policy API.
	// Optional. If not specified, default to "kubernetespolicy.googleapis.com:443".
	PolicyAPIAddress string `json:"POLICY_API_ADDRESS,omitempty"`
}

// Empty returns true if the config does not have necessary fields set and
// should be treated as empty.
func (g *GCPConfig) Empty() bool {
	return g.OrgID == "" && g.PrivateKeyFilename == ""
}

// Config contains the configuration for the installer.  The install process
// is made based on this configuration.
// +k8s:deepcopy-gen=true
type Config struct {
	// The user account that will drive the installation.  Required to insert
	// cluster administration role bindings into GKE clusters.
	User string `json:"user,omitempty"`

	// Contexts contains the names of the cluster contexts to attempt to install
	// into.
	Contexts []string `json:"contexts,omitempty"`

	// Git contains configuration specific to importing policies from a Git repo.
	// Git and GCP fields are mutually exclusive.
	Git GitConfig `json:"git,omitempty"`

	// GCP contains configuration specific to importing policies from Google Cloud Platform.
	// Git and GCP fields are mutually exclusive.
	GCP GCPConfig `json:"gcp,omitempty"`
}

// NewDefaultConfig creates a new Config struct with default values set.
func NewDefaultConfig() Config {
	c := Config{
		Git: GitConfig{
			SyncWaitSeconds: defaultSyncWaitTimeoutSeconds,
			SyncBranch:      defaultSyncBranch,
			// Default to expecting an ssh url since we also do some checking to see
			// if it's an ssh url during validation.
			UseSSH: true,
		},
	}
	return c
}

func expandHome(text string) string {
	text = strings.Replace(text, "$HOME", "/home/user", 1)
	userHomeOnHost := os.Getenv(homeOnHostEnv)
	if userHomeOnHost != "" {
		text = strings.Replace(text, userHomeOnHost, "/home/user", 1)
	}
	return text
}

// ExpandVarsCopy makes a copy of c, expanding path variables like $HOME with
// actual file paths.
func (c Config) ExpandVarsCopy() Config {
	newc := c
	newc.Git.KnownHostsFilename = expandHome(newc.Git.KnownHostsFilename)
	newc.Git.PrivateKeyFilename = expandHome(newc.Git.PrivateKeyFilename)
	newc.Git.CookieFilename = expandHome(newc.Git.CookieFilename)
	newc.GCP.PrivateKeyFilename = expandHome(newc.GCP.PrivateKeyFilename)
	return newc
}

// Load loads configuration from a reader in either YAML or JSON format.
func Load(r io.Reader) (Config, error) {
	b, err := ioutil.ReadAll(r)
	if err != nil {
		return Config{}, errors.Wrapf(err, "while reading config")
	}
	c := NewDefaultConfig()
	if err := yaml.Unmarshal(b, &c, yaml.DisallowUnknownFields); err != nil {
		return Config{}, errors.Wrapf(err, "while loading configuration")
	}
	return c, nil
}

// WriteInto writes the configuration into supplied writer in JSON format.
func (c Config) WriteInto(w io.Writer) error {
	b, err := yaml.Marshal(c)
	if err != nil {
		return errors.Wrapf(err, "while marshalling config")
	}
	i, err := w.Write(b)
	if i != len(b) {
		return errors.Errorf("short write")
	}
	return err
}

// Validate runs validations on the fields in Config.
func (c Config) Validate(exists FileExists) error {
	rc := c.ExpandVarsCopy()
	if !rc.Git.Empty() && !rc.GCP.Empty() {
		return errors.Errorf("gcp and git fields are mutually exclusive. Only one can be specified.")
	}
	if !rc.Git.Empty() {
		if rc.Git.SyncRepo == "" {
			return errors.Errorf("GIT_SYNC_REPO must be specified")
		}
		if rc.Git.UseSSH {
			if !rc.repoIsSSHURL() {
				return errors.Errorf("GIT_SYNC_SSH is set to true but GIT_SYNC_REPO is not using SSH protocol")
			}
			if rc.Git.PrivateKeyFilename == "" {
				return errors.Errorf("PRIVATE_KEY_FILE must be specified when GIT_SYNC_SSH is set to true")
			}
			if !exists.Check(rc.Git.PrivateKeyFilename) {
				return errors.Errorf("SSH private key doesn't exist: %v", rc.Git.PrivateKeyFilename)
			}
		} else {
			if rc.repoIsSSHURL() {
				return errors.Errorf("GIT_SYNC_SSH must be set to true since GIT_SYNC_REPO is using SSH protocol")
			}
		}
	} else if !rc.GCP.Empty() {
		if rc.GCP.OrgID == "" {
			return errors.Errorf("ORG_ID must be specified")
		}
		if rc.GCP.PrivateKeyFilename == "" {
			return errors.Errorf("PRIVATE_KEY_FILE must be specified")
		}
		if !exists.Check(rc.GCP.PrivateKeyFilename) {
			return errors.Errorf("private key doesn't exist: %v", rc.GCP.PrivateKeyFilename)
		}
	} else {
		return errors.Errorf("must set either 'git' or 'gcp'")
	}
	return nil
}

func (c Config) repoIsSSHURL() bool {
	for _, prefix := range []string{"ssh://", "git@"} {
		if strings.HasPrefix(c.Git.SyncRepo, prefix) {
			return true
		}
	}
	return false
}

// FileExists is an interface used to stub out file existence checks in unit
// tests.
type FileExists interface {
	// Check returns true if a file exists.
	Check(filename string) bool
}

// OsFileExists checks if files exists.
type OsFileExists struct{}

// Check implements FileExists.
func (OsFileExists) Check(filename string) bool {
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		return false
	}
	return true
}
