package nomostest

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/google/nomos/pkg/api/configmanagement"
	"github.com/google/nomos/pkg/reconcilermanager/controllers"
)

const gitServerSecret = "ssh-pub"
const namespaceSecret = "ssh-key"

func sshDir(nt *NT) string {
	nt.T.Helper()
	return filepath.Join(nt.TmpDir, "ssh")
}

func privateKeyPath(nt *NT) string {
	nt.T.Helper()
	return filepath.Join(sshDir(nt), "id_rsa.nomos")
}

func publicKeyPath(nt *NT) string {
	nt.T.Helper()
	return filepath.Join(sshDir(nt), "id_rsa.nomos.pub")
}

// createSSHKeySecret generates a public/public key pair for the test.
func createSSHKeyPair(nt *NT) {
	err := os.MkdirAll(sshDir(nt), fileMode)
	if err != nil {
		nt.T.Fatal("creating ssh directory:", err)
	}

	// ssh-keygen -t rsa -b 4096 -N "" \
	//   -f /opt/testing/nomos/id_rsa.nomos
	//   -C "key generated for use in e2e tests"
	out, err := exec.Command("ssh-keygen", "-t", "rsa", "-b", "4096", "-N", "",
		"-f", privateKeyPath(nt),
		"-C", "key generated for use in e2e tests").Output()
	if err != nil {
		nt.T.Log(string(out))
		nt.T.Fatal("generating rsa key for ssh:", err)
	}
}

// createSecret creates secret in the given namespace using 'keypath'.
func createSecret(nt *NT, namespace, name, keyPath string) {
	// kubectl create secret generic 'name' \
	//   -n='namespace' \
	//   --from-file='keyPath'
	nt.MustKubectl("create", "secret", "generic", name,
		"-n", namespace,
		"--from-file", keyPath)
}

// generateSSHKeys generates a public/public key pair for the test.
//
// It turns out kubectl create secret is annoying to emulate, and it doesn't
// expose the inner logic to outside consumers. So instead of trying to do it
// ourselves, we're shelling out to kubectl to ensure we create a valid set of
// secrets.
func generateSSHKeys(nt *NT) string {
	nt.T.Helper()

	createSSHKeyPair(nt)

	createSecret(nt, configmanagement.ControllerNamespace, controllers.GitCredentialVolume,
		fmt.Sprintf("ssh=%s", privateKeyPath(nt)))

	createSecret(nt, testGitNamespace, gitServerSecret,
		filepath.Join(publicKeyPath(nt)))

	return privateKeyPath(nt)
}

// downloadSSHKey downloads the private SSH key from Cloud Secret Manager.
func downloadSSHKey(nt *NT) string {
	dir := sshDir(nt)
	err := os.MkdirAll(dir, fileMode)
	if err != nil {
		nt.T.Fatal("creating ssh directory:", err)
	}

	out, err := exec.Command("gcloud", "secrets", "versions", "access", "latest", "--secret=config-sync-ci-ssh-private-key").CombinedOutput()
	if err != nil {
		nt.T.Log(string(out))
		nt.T.Fatal("downloading SSH key:", err)
	}

	if err := ioutil.WriteFile(privateKeyPath(nt), out, 0600); err != nil {
		nt.T.Fatal("saving SSH key:", err)
	}

	createSecret(nt, configmanagement.ControllerNamespace, controllers.GitCredentialVolume,
		fmt.Sprintf("ssh=%s", privateKeyPath(nt)))

	return privateKeyPath(nt)
}

// CreateNamespaceSecret creates secret in a given namespace using privateKeyPath.
func CreateNamespaceSecret(nt *NT, ns string) {
	nt.T.Helper()
	privateKeypath := nt.gitPrivateKeyPath
	if len(privateKeypath) == 0 {
		privateKeypath = privateKeyPath(nt)
	}
	createSecret(nt, ns, namespaceSecret, fmt.Sprintf("ssh=%s", privateKeypath))
}
