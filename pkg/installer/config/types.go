// Package config contains the shared configuration for the installer script.
//
// The top level struct is Config.  It includes all other types defined in this
// package.
//
// Deprecation notice:
// The installer script will be replaced with an operator-based approach that
// replaces custom installation logic with cluster-side installation operator
// process and a static configuration file that can be applied using
//
//     kubectl -f apply ...
//
// All configuration fields that are not explicitly annotated with a version
// should be assumed to be "v1alpha0" of this configuration.  All fields added
// thereafter should be annotated with the current version like so:
//
//     type Config struct {
//        // ... old fields
//
//        // NewField is new.
//        // Since: v1alpha2
//        NewField string
//     }
//
// This is not full K8S style versioning approach, but should work for us given
// the pending retirement of this code.  In case this proves inadequate, we can
// retrofit versioning by instantiating proper external API objects.
package config

import (
	"io"
	"io/ioutil"
	"os"
	"strings"

	// TODO(filmil): Revisit once https://github.com/ghodss/yaml has a new
	// release that includes the closed pull request
	// https://github.com/ghodss/yaml/pull/27
	"github.com/filmil/yaml"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	defaultSyncWaitTimeoutSeconds = 15
	defaultSyncBranch             = "master"

	// Version is the current version of the configuration.  This is specified
	// informally since it's a bit of an overkill to declare the API
	// specifically for the configuration that is pending retirement.
	Version = "v1alpha1"

	// APIGroup is a group marker for the installer configuration.  This is not
	// used, except for documentation.
	APIGroup = "installer.nomos.dev"
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

// Cluster defines a mapping from a cluster context to an installation-wide
// cluster name.
//
// Introduced since Config v1alpha1.
type Cluster struct {
	// ClusterName is the installation-wide cluster name.
	Name string `json:"name"`

	// Context is the name of the local kubectl context used for a cluster.
	// This field has exactly the same semantics as the Config.Contexts entry.
	Context string `json:"context"`
}

// Config contains the configuration for the installer.  The install process
// is made based on this configuration.
// At v1alpha1, we are adopting a versioning discipline.  Any fields that are
// not annotated are assumed to come from v1alpha0.
// +k8s:deepcopy-gen=true
type Config struct {
	// Allow optionally specifying type metadata.  While this is not strictly
	// needed, let's adopt a versioning discipline until the installer is
	// retired in favor of the nomos operator.
	metav1.TypeMeta `json:",inline"`

	// The user account that will drive the installation.  Required to insert
	// cluster administration role bindings into GKE clusters.
	User string `json:"user,omitempty"`

	// Contexts contains the names of the cluster contexts to attempt to install
	// into.
	// This field is discarded if Clusters (below) is defined, and
	// element-wise replaced with the content of Clusters.Context.
	Contexts []string `json:"contexts,omitempty"`

	// Git contains configuration specific to importing policies from a Git repo.
	// Git and GCP fields are mutually exclusive.
	Git GitConfig `json:"git,omitempty"`

	// GCP contains configuration specific to importing policies from Google Cloud Platform.
	// Git and GCP fields are mutually exclusive.
	GCP GCPConfig `json:"gcp,omitempty"`

	// Clusters is a list of name-context pairs used during installation.  It
	// fully replaces the information from Contexts above if it is supplied.
	// Since: v1alpha1
	Clusters []Cluster `json:"clusters,omitempty"`
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
	return strings.Replace(text, "$HOME", os.Getenv("HOME"), 1)
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
	// Make the objects usable for v1alpha1.
	if len(c.Clusters) > 0 {
		// Trample over Contexts with the data extracted from c.Clusters
		c.Contexts = []string{}
		for _, cl := range c.Clusters {
			c.Contexts = append(c.Contexts, cl.Context)
		}
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
