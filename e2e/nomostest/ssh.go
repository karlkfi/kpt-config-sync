package nomostest

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/google/nomos/pkg/api/configmanagement"
)

// generateSSHKeys generates a public/public key pair for the test.
//
// It turns out kubectl create secret is annoying to emulate, and it doesn't
// expose the inner logic to outside consumers. So instead of trying to do it
// ourselves, we're shelling out to kubectl to ensure we create a valid set of
// secrets.
func generateSSHKeys(nt *NT, kcfg string) string {
	nt.T.Helper()

	sshDir := filepath.Join(nt.TmpDir, "ssh")
	err := os.MkdirAll(sshDir, fileMode)
	if err != nil {
		nt.T.Fatal("creating ssh directory:", err)
	}

	privateKeyPath := filepath.Join(sshDir, "id_rsa.nomos")
	publicKeyPath := filepath.Join(sshDir, "id_rsa.nomos.pub")

	// ssh-keygen -t rsa -b 4096 -N "" \
	//   -f /opt/testing/nomos/id_rsa.nomos
	//   -C "key generated for use in e2e tests"
	out, err := exec.Command("ssh-keygen", "-t", "rsa", "-b", "4096", "-N", "",
		"-f", privateKeyPath,
		"-C", "key generated for use in e2e tests").Output()
	if err != nil {
		nt.T.Log(string(out))
		nt.T.Fatal("generating rsa key for ssh:", err)
	}

	// kubectl create secret generic git-creds \
	//   -n=config-management-system \
	//   --from-file=ssh="${TEST_DIR}/id_rsa.nomos"
	nt.Kubectl("create", "secret", "generic", "git-creds",
		"-n", configmanagement.ControllerNamespace,
		"--from-file", fmt.Sprintf("ssh=%s", privateKeyPath))

	// kubectl create secret generic ssh-pub \
	//   -n="${GIT_SERVER_NS}" \
	//   --from-file=/opt/testing/nomos/id_rsa.nomos.pub
	nt.Kubectl("create", "secret", "generic", "ssh-pub",
		"-n", testGitNamespace,
		"--from-file", filepath.Join(publicKeyPath))
	return privateKeyPath
}
