package nomostest

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"

	"github.com/google/nomos/pkg/api/configmanagement"
	"github.com/google/nomos/pkg/core"
	"golang.org/x/crypto/ssh"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// generateSSHKeys generates a public/public key pair for the test.
func generateSSHKeys(nt *NT) []core.Object {
	nt.T.Helper()

	// Generate the private key for the git-sync Pod.
	privateKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		nt.T.Fatal(err)
	}
	privateKeyBytes := x509.MarshalPKCS1PrivateKey(privateKey)
	privateKeyString := base64.StdEncoding.EncodeToString(privateKeyBytes)
	privateSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "git-creds",
			Namespace: configmanagement.ControllerNamespace,
		},
		Data: map[string][]byte{
			"ssh": []byte(privateKeyString),
		},
		Type: corev1.SecretTypeOpaque,
	}

	// Generate the public key for git-server.
	publicKey, err := ssh.NewPublicKey(privateKey.Public())
	if err != nil {
		nt.T.Fatalf("generating public key: %v", err)
	}
	publicKeyBytes := ssh.MarshalAuthorizedKey(publicKey)
	publicKeyString := base64.StdEncoding.EncodeToString(publicKeyBytes)
	publicSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ssh-pub",
			Namespace: testGitNamespace,
		},
		Data: map[string][]byte{
			"id_rsa.nomos.pub": []byte(publicKeyString),
		},
		Type: corev1.SecretTypeOpaque,
	}
	return []core.Object{privateSecret, publicSecret}
}
