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
)

// DefaultConfig contains the empty default values for a Config.
var DefaultConfig = Config{
	Git: GitConfig{
		SyncWaitSeconds: defaultSyncWaitTimeoutSeconds,
		SyncBranch:      defaultSyncBranch,
		// Default to expecting an ssh url since we also do some checking to see
		// if it's an ssh url during validation.
		UseSSH: true,
	},
}

// GitConfig contains the settings for the git importer repository.
type GitConfig struct {
	// SyncRepo is the git repository name to sync from.
	SyncRepo string `json:"GIT_SYNC_REPO"`

	// UseSSH is true if we are syncing the repo using ssh.
	UseSSH bool `json:"GIT_SYNC_SSH"`

	// PrivateKeyFilename is the filename containing the private key used
	// for SSH authentication.  This entry is communicated through a file
	// to avoid exposing the contents of the secret in the process table.
	// If this entry is empty, SSH will be set to false.
	PrivateKeyFilename string `json:"PRIVATE_KEY_FILENAME,omitempty"`

	// KnownHostsFilename is the filename containing the known hosts SSH
	// file that Kubernetes will use.  Not copied if not defined.
	KnownHostsFilename string `json:"KNOWN_HOSTS_FILENAME,omitempty"`

	// SyncBranch is the branch to sync from.  Default: "master".
	SyncBranch string `json:"GIT_SYNC_BRANCH"`

	// RootPolicyDir is the absolute path of the directory that contains
	// the local policy.  Default: the root directory of the repo.
	RootPolicyDir string `json:"POLICY_DIR"`

	// SyncWaitSeconds is the time duration in seconds between consecutive
	// syncs.  Default: 15 seconds.
	SyncWaitSeconds int64 `json:"GIT_SYNC_WAIT"`
}

// Empty returns true if the config does not have necessary fields set and
// should be treated as empty.
func (g *GitConfig) Empty() bool {
	return g.SyncRepo == "" && g.RootPolicyDir == ""
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

	// Git contains the git-specific configuration.
	Git GitConfig `json:"git,omitempty"`
}

// NewDefaultConfig creates a new Config struct with default values set
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
	c.Git.KnownHostsFilename = strings.Replace(c.Git.KnownHostsFilename, "$HOME", "/home/user", 1)
	c.Git.PrivateKeyFilename = strings.Replace(c.Git.PrivateKeyFilename, "$HOME", "/home/user", 1)
	return c, nil
}

// WriteInto writes the configuration into supplied writer in JSON format.
func (c Config) WriteInto(w io.Writer) error {
	c.Git.KnownHostsFilename = strings.Replace(c.Git.KnownHostsFilename, "/home/user", "$HOME", 1)
	c.Git.PrivateKeyFilename = strings.Replace(c.Git.PrivateKeyFilename, "/home/user", "$HOME", 1)
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
	if !c.Git.Empty() {
		if c.Git.SyncRepo == "" {
			return errors.Errorf("git repo not specified")
		}
		if c.Git.UseSSH {
			if !c.repoIsSSHURL() {
				return errors.Errorf("ssh specified for non-ssh git repo url")
			}
			if c.Git.PrivateKeyFilename == "" {
				return errors.Errorf("ssh specified for git repo, but private key not specified")
			}
			if !exists.Check(c.Git.PrivateKeyFilename) {
				return errors.Errorf("ssh specified for git repo, but private key doesn't exist: %v", c.Git.PrivateKeyFilename)
			}
		} else {
			if c.repoIsSSHURL() {
				return errors.Errorf("ssh specified as disabled for ssh git repo url")
			}
		}
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
