package nomostest

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	"github.com/google/nomos/e2e/nomostest/testing"
	"github.com/google/nomos/pkg/importer/filesystem"
	"github.com/google/nomos/pkg/syncer/reconcile"
	"github.com/google/nomos/pkg/testing/fake"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
)

const (
	// remoteName is static as every git repository has exactly one remote.
	remoteName = "origin"
	// MainBranch is static as behavior when switching branches is never under
	// test.
	MainBranch = "main"

	// SafetyNSPath is the path to the safety namespace yaml file.
	SafetyNSPath = "acme/namespaces/safety/ns.yaml"

	// SafetyNS is the name of the safety namespace.
	SafetyNS = "safety"
)

// Repository is a local git repository with a connection to a repository
// on the git-server for the test.
//
// We shell out for git commands as the git libraries are difficult to configure
// ssh for, and git-server requires ssh authentication.
type Repository struct {
	// Root is the location on the machine running the test at which the local
	// repository is stored.
	Root string
	// Format is the source format for parsing the repository (hierarchy or
	// unstructured).
	Format filesystem.SourceFormat

	T testing.NTB

	// Name is the local repository name.
	Name string

	// RemoteRepoName is the name of the remote repository.
	RemoteRepoName string

	// RemoteURL is the remote URL of the repository.
	RemoteURL string
}

// NewRepository creates a remote repo on the git provider.
// Locally, it writes the repository to `tmpdir`/repos/`name`.
//
// For root repo, `name` is `sot.git`.
// For namespace repo, `name` is the name of the namespace.
func NewRepository(nt *NT, name string, sourceFormat filesystem.SourceFormat) *Repository {
	nt.T.Helper()

	localDir := filepath.Join(nt.TmpDir, "repos", name)

	g := &Repository{
		Root:   localDir,
		Format: sourceFormat,
		T:      nt.T,
		Name:   name,
	}

	repoName, err := nt.GitProvider.CreateRepository(name)
	// Add the repo to nt.RemoteRepositories immediately after it is created to reuse the repo.
	nt.RemoteRepositories[name] = g
	if err != nil {
		nt.T.Fatal(err)
	}
	g.RemoteRepoName = repoName
	g.RemoteURL = nt.GitProvider.RemoteURL(nt.gitRepoPort, repoName)

	g.init(nt.gitPrivateKeyPath)
	g.initialCommit(sourceFormat)

	return g
}

// ReInit re-initializes the repo to the initial state.
func (g *Repository) ReInit(nt *NT, sourceFormat filesystem.SourceFormat) {
	nt.T.Helper()

	g.init(nt.gitPrivateKeyPath)
	g.initialCommit(sourceFormat)
}

func (g *Repository) gitCmd(command ...string) *exec.Cmd {
	// The -C flag executes git from repository root.
	// https://git-scm.com/docs/git#Documentation/git.txt--Cltpathgt
	args := []string{"-C", g.Root}
	args = append(args, command...)
	return exec.Command("git", args...)
}

// Git wraps shelling out to git, ensuring we're running from the git repository
//
// Fails immediately if any git command fails.
func (g *Repository) Git(command ...string) {
	g.T.Helper()

	cmd := g.gitCmd(command...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		g.T.Log(string(out))
		g.T.Fatal(err)
	}
}

// initialCommit initializes the Nomos repo with the Repo object.
func (g *Repository) initialCommit(sourceFormat filesystem.SourceFormat) {
	g.T.Helper()

	// Add the README to the inside of acme/ so the directory is guaranteed to
	// exist - ACM refuses to sync to non-existent directories, and git requires
	// at least one file in order for a directory to exist.
	g.AddFile("acme/README.md", []byte("Test repository."))
	if g.Name == rootRepo {
		// Keep a safety namespace to avoid failing the safety check.
		g.Add(SafetyNSPath, fake.NamespaceObject(SafetyNS))
	}
	switch sourceFormat {
	case filesystem.SourceFormatHierarchy:
		// Hierarchy format requires a Repo object.
		g.Add("acme/system/repo.yaml", fake.RepoObject())
	case filesystem.SourceFormatUnstructured:
		// It is an error for unstructured repos to include the Repo object.
	default:
		g.T.Fatalf("Unrecognized SourceFormat: %q", sourceFormat)
	}
	g.CommitAndPush("initial commit")
}

// init initializes this git repository and configures it to talk to the cluster
// under test.
func (g *Repository) init(privateKey string) {
	g.T.Helper()

	if err := os.RemoveAll(g.Root); err != nil {
		g.T.Fatal(err)
	}

	err := os.MkdirAll(g.Root, fileMode)
	if err != nil {
		g.T.Fatal(err)
	}
	g.Git("init")
	g.Git("checkout", "-b", MainBranch)

	// We have to configure username/email or else committing to the repository
	// produces errors.
	g.Git("config", "user.name", "E2E Testing")
	g.Git("config", "user.email", "nomos-team@google.com")

	// Use ssh rather than the default that git uses, as the default does not know
	// how to use private key files.
	g.Git("config", "ssh.variant", "ssh")
	// Overwrite the ssh command to:
	// 1) Not perform host key checking for git-server, since this isn't set up
	//   properly and we don't care.
	// 2) Use the private key file we generated.
	g.Git("config", "core.sshCommand",
		fmt.Sprintf("ssh -q -o StrictHostKeyChecking=no -i %s", privateKey))
	// Point the origin remote.
	g.Git("remote", "add", remoteName, g.RemoteURL)
}

// Add writes a YAML or JSON representation of obj to `path` in the git
// repository, and `git add`s the file. Does not commit/push.
//
// Overwrites the file if it already exists.
// Automatically writes YAML or JSON based on the path's extension.
//
// Don't put multiple manifests in the same file unless parsing multi-manifest
// files is the behavior under test. In that case, use AddFile.
func (g *Repository) Add(path string, obj client.Object) {
	g.T.Helper()
	AddTestLabel(obj)
	// TODO(willbeason): Figure out how to cleanly inject runtime.Scheme here.

	// We have to make a pass through json since yaml.Marshal does not respect
	// json "omitempty" directives.
	var bytes []byte
	var err error
	var u *unstructured.Unstructured
	ext := filepath.Ext(path)
	switch ext {
	case ".yaml", ".yml":
		// We must convert through JSON/Unstructured to avoid "omitempty" fields
		// from being specified.
		u, err = reconcile.AsUnstructuredSanitized(obj)
		if err != nil {
			g.T.Fatal(err)
		}
		bytes, err = yaml.Marshal(u)
	case ".json":
		u, err = reconcile.AsUnstructuredSanitized(obj)
		if err != nil {
			g.T.Fatal(err)
		}
		bytes, err = json.MarshalIndent(u, "", "  ")
	default:
		// If you're seeing this error, use "AddFile" instead to test ignoring
		// files with extensions we ignore.
		err = fmt.Errorf("invalid extension to write object to, %q, use .AddFile() instead", ext)
	}
	if err != nil {
		g.T.Fatal(err)
	}

	g.AddFile(path, bytes)
}

// AddFile writes `bytes` to `file` in the git repository.
// This function should only be directly used for testing the literal YAML/JSON
// parsing logic.
//
// Path is relative to the Git repository root.
// Overwrites `file` if it already exists.
// Does not commit/push.
func (g *Repository) AddFile(path string, bytes []byte) {
	g.T.Helper()

	absPath := filepath.Join(g.Root, path)

	err := os.MkdirAll(filepath.Dir(absPath), fileMode)
	if err != nil {
		g.T.Fatal(err)
	}

	// Write bytes to file.
	err = ioutil.WriteFile(absPath, bytes, fileMode)
	if err != nil {
		g.T.Fatal(err)
	}
	// Add the file to Git.
	g.Git("add", absPath)
}

// CopyDirectory copies the directory from source to destination.
// Overwrites the directory if it already exists.
// Does not commit/push.
func (g *Repository) CopyDirectory(sourceDir, destDir string) {
	g.T.Helper()

	absDestPath := filepath.Join(g.Root, destDir)
	if _, err := exec.Command("cp", "-r", sourceDir, absDestPath).CombinedOutput(); err != nil {
		g.T.Fatal(err)
	}
	// Add the directory to Git.
	g.Git("add", absDestPath)
}

// Remove deletes `file` from the git repository.
// If `file` is a directory, deletes the directory.
// Returns error if the file does not exist.
// Does not commit/push.
func (g *Repository) Remove(path string) {
	g.T.Helper()

	absPath := filepath.Join(g.Root, path)

	err := os.RemoveAll(absPath)
	if err != nil {
		g.T.Fatal(err)
	}

	g.Git("add", absPath)
}

// CommitAndPush commits any changes to the git repository, and
// pushes them to the git server.
// We don't care about differentiating between committing and pushing
// for tests.
func (g *Repository) CommitAndPush(msg string) {
	g.T.Helper()
	g.CommitAndPushBranch(msg, MainBranch)
}

// CommitAndPushBranch commits any changes to the git branch, and
// pushes them to the git server.
func (g *Repository) CommitAndPushBranch(msg, branch string) {
	g.T.Helper()

	g.Git("commit", "-m", msg)

	g.T.Logf("[repo %s] committing %q (%s)", path.Base(g.Root), msg, g.Hash())
	g.Git("push", "-f", "-u", remoteName, branch)
}

// CreateBranch creates and checkouts a new branch at once.
func (g *Repository) CreateBranch(branch string) {
	g.T.Helper()

	g.Git("branch", branch)
	g.CheckoutBranch(branch)
}

// CheckoutBranch checkouts a branch.
func (g *Repository) CheckoutBranch(branch string) {
	g.T.Helper()

	g.Git("checkout", branch)
}

// RenameBranch renames the current branch with a new one both locally and remotely.
// The old branch will be deleted from remote.
func (g *Repository) RenameBranch(current, new string) {
	g.T.Helper()

	g.Git("branch", "-m", current, new)
	g.Git("push", remoteName, "-u", new)
	g.Git("push", remoteName, "--delete", current)
}

// Hash returns the current hash of the git repository.
//
// Immediately ends the test on error.
func (g *Repository) Hash() string {
	// Get the hash of the git repository.
	// git rev-parse --verify HEAD
	out, err := g.gitCmd("rev-parse", "--verify", "HEAD").CombinedOutput()
	if err != nil {
		g.T.Log(string(out))
		g.T.Fatal(err)
	}
	return strings.TrimSpace(string(out))
}
