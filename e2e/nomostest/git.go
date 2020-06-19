package nomostest

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/testing/fake"
	"sigs.k8s.io/yaml"
)

const (
	remoteName = "origin"
	// TODO(willbeason): Allow configuring branch.
	branchName = "master"
)

// Repository is a local git repository with a connection to a repository
// on the git-server for the test.
//
// We shell out for git commands as the git libraries are difficult to configure
// ssh for, and git-server requires ssh authentication.
type Repository struct {
	// root is the location on the machine running the test at which the local
	// repository is stored.
	root string

	T *testing.T
}

// NewRepository creates a repository named `name`, that connects to git-server
// via port `port`.
//
// Writes the repository to `tmpdir`/repos/`name`. Repositories in the same
// test must have unique names.
//
// For now, `name` must be "sot.git" until we support dynamically creating
// repositories.
func NewRepository(t *testing.T, name, tmpDir, privateKeyPath string, port int) *Repository {
	t.Helper()

	localDir := filepath.Join(tmpDir, "repos", name)

	g := &Repository{
		root: localDir,
		T:    t,
	}
	g.init(name, privateKeyPath, port)

	g.initialCommit()

	return g
}

func (g *Repository) gitCmd(command ...string) *exec.Cmd {
	// The -C flag executes git from repository root.
	// https://git-scm.com/docs/git#Documentation/git.txt--Cltpathgt
	args := []string{"-C", g.root}
	args = append(args, command...)
	return exec.Command("git", args...)
}

// git wraps shelling out to git, ensuring we're running from the git repository
//
// Fails immediately if any git command fails.
func (g *Repository) git(command ...string) {
	g.T.Helper()

	cmd := g.gitCmd(command...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		g.T.Log(string(out))
		g.T.Fatal(err)
	}
}

// initialCommit initializes the Nomos repo with the Repo object.
func (g *Repository) initialCommit() {
	// TODO(willbeason): Allow initializing an unstructured repo.
	g.T.Helper()

	g.AddFile("README.md", []byte("Test repository."))
	g.Add("acme/system/repo.yaml", fake.RepoObject())
	g.CommitAndPush("initial commit")
}

// init initializes this git repository and configures it to talk to the cluster
// under test.
func (g *Repository) init(name, privateKey string, port int) {
	g.T.Helper()

	err := os.MkdirAll(g.root, fileMode)
	if err != nil {
		g.T.Fatal(err)
	}
	g.git("init")

	// We have to configure username/email or else committing to the repository
	// produces errors.
	g.git("config", "user.name", "E2E Testing")
	g.git("config", "user.email", "nomos-team@google.com")

	// Use ssh rather than the default that git uses, as the default does not know
	// how to use private key files.
	g.git("config", "ssh.variant", "ssh")
	//Overwrite the ssh command to:
	// 1) Not perform host key checking for git-server, since this isn't set up
	//   properly and we don't care.
	// 2) Use the private key file we generated.
	g.git("config", "core.sshCommand",
		fmt.Sprintf("ssh -q -o StrictHostKeyChecking=no -i %s", privateKey))
	// Point the origin remote at the port we've forwarded to git-server.
	g.git("remote", "add", remoteName,
		fmt.Sprintf("ssh://git@localhost:%d/git-server/repos/%s", port, name))
}

// Add writes a YAML or JSON representation of obj to `path` in the git
// repository, and `git add`s the file. Does not commit/push.
//
// Overwrites the file if it already exists.
// Automatically writes YAML or JSON based on the path's extension.
//
// Don't put multiple manifests in the same file unless parsing multi-manifest
// files is the behavior under test. In that case, use AddFile.
func (g *Repository) Add(path string, obj core.Object) {
	g.T.Helper()

	// We have to make a pass through json since yaml.Marshal does not respect
	// json "omitempty" directives.
	var bytes []byte
	var err error
	ext := filepath.Ext(path)
	switch ext {
	case ".yaml", ".yml":
		bytes, err = yaml.Marshal(obj)
	case ".json":
		bytes, err = json.MarshalIndent(obj, "", "  ")
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

	absPath := filepath.Join(g.root, path)

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
	g.git("add", absPath)
}

// Remove deletes `file` from the git repository.
// If `file` is a directory, deletes the directory.
// Returns error if the file does not exist.
// Does not commit/push.
func (g *Repository) Remove(path string) {
	g.T.Helper()

	err := os.Remove(path)
	if err != nil {
		g.T.Fatal(err)
	}

	g.git("add", filepath.Join(g.root, path))
}

// CommitAndPush commits any changes to the git repository, and
// pushes them to the git server.
// We don't care about differentiating between committing and pushing
// for tests.
func (g *Repository) CommitAndPush(msg string) {
	g.T.Helper()

	g.git("commit", "-m", msg)

	g.T.Logf("committing %q", msg)
	g.git("push", "-u", remoteName, branchName)
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
