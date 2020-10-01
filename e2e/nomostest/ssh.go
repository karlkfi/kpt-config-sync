package nomostest

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/google/nomos/pkg/api/configmanagement"
)

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
	nt.Kubectl("create", "secret", "generic", name,
		"-n", namespace,
		"--from-file", keyPath)
}

// generateSSHKeys generates a public/public key pair for the test.
//
// It turns out kubectl create secret is annoying to emulate, and it doesn't
// expose the inner logic to outside consumers. So instead of trying to do it
// ourselves, we're shelling out to kubectl to ensure we create a valid set of
// secrets.
func generateSSHKeys(nt *NT, kcfg string) string {
	nt.T.Helper()

	createSSHKeyPair(nt)

	createSecret(nt, configmanagement.ControllerNamespace, "git-creds",
		fmt.Sprintf("ssh=%s", privateKeyPath(nt)))

	createSecret(nt, testGitNamespace, "ssh-pub",
		filepath.Join(publicKeyPath(nt)))

	return privateKeyPath(nt)
}

// createNamespaceSecret creates secret in a given namespace using privateKeyPath.
func createNamespaceSecret(nt *NT, ns string) {
	nt.T.Helper()
	createSecret(nt, ns, "ssh-key", fmt.Sprintf("ssh=%s", privateKeyPath(nt)))
}
