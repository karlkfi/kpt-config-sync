package kubectl

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"testing"

	"github.com/blang/semver"
	"github.com/google/go-cmp/cmp"
	"github.com/google/nomos/pkg/client/meta/fake"
	"github.com/google/nomos/pkg/client/restconfig"
	"github.com/google/nomos/pkg/process/exec"
	"github.com/pkg/errors"
	"k8s.io/api/admissionregistration/v1beta1"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/api/core/v1"
	api_errors "k8s.io/apimachinery/pkg/api/errors"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestClusterList(t *testing.T) {
	// These tests do not need meta client, turn it off.
	useMetaClient = false
	tests := []struct {
		name       string
		configText string
		expected   ClusterList
		err        error
	}{
		{
			name: "Basic",
			expected: ClusterList{
				Clusters: map[string]string{},
				Current:  "",
			},
		},
		{
			name: "OneConfig",
			expected: ClusterList{
				Clusters: map[string]string{
					"dev-frontend": "development",
					"exp-scratch":  "scratch",
				},
				Current: "dev-frontend",
			},
			configText: `` +
				`apiVersion: v1
kind: Config
preferences: {}
clusters:
- cluster:
  name: development
- cluster:
  name: scratch
users:
- name: developer
- name: experimenter
contexts:
- context:
    cluster: development
  name: dev-frontend
- context:
    cluster: scratch
  name: exp-scratch
current-context: dev-frontend
`,
		},
		{
			name: "Unparseable config",
			expected: ClusterList{
				Clusters: map[string]string{
					"dev-frontend": "development",
					"exp-scratch":  "scratch",
				},
				Current: "dev-frontend",
			},
			configText: "the_unparseable_config",
			err:        errors.Errorf("cannot unmarshal string"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// TempDir is writable in the build container.
			d, err := ioutil.TempDir("", "home")
			if err != nil {
				t.Fatalf("could not create temp directory: %v", err)
			}
			defer os.Remove(d)
			// Replacement for user.Current() which is not usable without CGO.
			restconfig.SetCurrentUserForTest(
				&user.User{
					Uid:      "0",
					Username: "nobody",
					HomeDir:  filepath.Join(d, "nobody")}, nil)
			err = os.MkdirAll(filepath.Join(d, "nobody/.kube"), os.ModeDir|os.ModePerm)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			cfg, err := os.Create(filepath.Join(d, "nobody/.kube/config"))
			if err != nil {
				t.Fatalf("could not open config: %v", err)
			}
			defer os.Remove(cfg.Name())
			fmt.Fprint(cfg, tt.configText)
			err = cfg.Close()
			if err != nil {
				t.Fatalf("could not close config: %v", err)
			}
			cl, err := LocalClusters()
			if err != nil {
				if tt.err != nil {
					if !strings.ContainsAny(tt.err.Error(), err.Error()) {
						t.Errorf("wront error: %q, want: %q", err.Error(), tt.err.Error())
					}
				} else {
					t.Errorf("unexpected error: %v", err)
				}

				return
			}
			if !cmp.Equal(tt.expected, cl) {
				t.Errorf("LocalClusters:()\n%#v,\nwant:\n%#v,\ndiff:\n%v",
					cl, tt.expected, cmp.Diff(cl, tt.expected))
			}
		})
	}
}

func TestVersion(t *testing.T) {
	// These tests do not need meta client, turn it off.
	useMetaClient = false
	// Replacement for user.Current() which is not usable without CGO.
	restconfig.SetCurrentUserForTest(&user.User{Uid: "0", Username: "nobody"}, nil)
	tests := []struct {
		name     string
		output   string
		expected semver.Version
		err      error
	}{
		{
			name: "Simple version",
			output: `{
  "clientVersion": {
    "major": "1",
    "minor": "8",
    "gitVersion": "v1.8.6",
    "gitCommit": "6260bb08c46c31eea6cb538b34a9ceb3e406689c",
    "gitTreeState": "clean",
    "buildDate": "2017-12-21T06:34:11Z",
    "goVersion": "go1.8.3",
    "compiler": "gc",
    "platform": "linux/amd64"
  },
  "serverVersion": {
    "major": "1",
    "minor": "9",
    "gitVersion": "v1.9.1",
    "gitCommit": "3a1c9449a956b6026f075fa3134ff92f7d55f812",
    "gitTreeState": "clean",
    "buildDate": "2018-01-04T11:40:06Z",
    "goVersion": "go1.9.2",
    "compiler": "gc",
    "platform": "linux/amd64"
  }
}
		`,
			expected: semver.MustParse("1.9.1"),
		},
		{
			name: "Complex semver",
			output: `{
  "clientVersion": {
  },
  "serverVersion": {
    "gitVersion": "v1.9.2-rc.alpha.something.other+dirty"
  }
}
		`,
			expected: semver.MustParse("1.9.2-rc.alpha.something.other+dirty"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exec.SetFakeOutputsForTest(strings.NewReader(tt.output), nil, nil)
			c := New(context.Background())
			actual, err := c.GetClusterVersion()
			if err != nil {
				if tt.err != nil && err.Error() != tt.err.Error() {
					t.Errorf("err.Error(): %v, want: %v", err, tt.err)
				} else {
					t.Errorf("unexpected error: %v", err)
				}
			}
			if actual.NE(tt.expected) {
				t.Errorf("actual: %v, want: %v", actual, tt.expected)
			}
		})
	}
}

func TestDeleteSecret(t *testing.T) {
	tests := []struct {
		name       string
		secretName string
		namespace  string
		outErr     error
	}{
		{
			name:       "basic positive",
			secretName: "someSecret",
			namespace:  "someNamespace",
			outErr:     nil,
		},
	}
	client := fake.NewClient()
	kc := NewWithClient(context.Background(), client)
	for _, test := range tests {
		s, err := client.Kubernetes().CoreV1().Secrets(test.namespace).Create(&v1.Secret{
			ObjectMeta: meta_v1.ObjectMeta{
				Name: test.secretName,
			}})
		if s == nil {
			t.Errorf("No secret created. Should have created %s", test.secretName)
		}
		if err != nil {
			t.Errorf("Error creating secret %s : %v", test.secretName, err)
		}
		s, err = client.Kubernetes().CoreV1().Secrets(test.namespace).Get(test.secretName, meta_v1.GetOptions{})
		if s == nil {
			t.Errorf("Secret %s not found", test.secretName)
		}
		if err != nil {
			t.Errorf("Error getting secret %s : %v", test.secretName, err)
		}
		err = kc.DeleteSecret(test.secretName, test.namespace)
		if err != test.outErr {
			t.Errorf("Expected %v returned from DeleteSecret, but got %v", test.outErr, err)
		}
		_, err = client.Kubernetes().CoreV1().Secrets(test.namespace).Get(test.secretName, meta_v1.GetOptions{})
		if !api_errors.IsNotFound(err) {
			t.Errorf("Deleted secret %v still exists", test.secretName)
		}
	}
}

func TestDeleteConfigmap(t *testing.T) {
	tests := []struct {
		name          string
		configMapName string
		namespace     string
		outErr        error
	}{
		{
			name:          "basic positive",
			configMapName: "aConfigMap",
			namespace:     "someNamespace",
			outErr:        nil,
		},
	}
	client := fake.NewClient()
	kc := NewWithClient(context.Background(), client)
	for _, test := range tests {
		cm, err := client.Kubernetes().CoreV1().ConfigMaps(test.namespace).Create(&v1.ConfigMap{
			ObjectMeta: meta_v1.ObjectMeta{
				Name: test.configMapName,
			}})
		if cm == nil {
			t.Errorf("No configmap created. Should have created %s", test.configMapName)
		}
		if err != nil {
			t.Errorf("Error creating configmap %s : %v", test.configMapName, err)
		}
		cm, err = client.Kubernetes().CoreV1().ConfigMaps(test.namespace).Get(test.configMapName, meta_v1.GetOptions{})
		if cm == nil {
			t.Errorf("ConfigMap %s not found", test.configMapName)
		}
		if err != nil {
			t.Errorf("Error getting configmap %s : %v", test.configMapName, err)
		}
		err = kc.DeleteConfigMap(test.configMapName, test.namespace)
		if err != test.outErr {
			t.Errorf("Expected %v returned from DeleteConfigMap, but got %v", test.outErr, err)
		}
		_, err = client.Kubernetes().CoreV1().ConfigMaps(test.namespace).Get(test.configMapName, meta_v1.GetOptions{})
		if !api_errors.IsNotFound(err) {
			t.Errorf("Deleted configmap %v still exists", test.configMapName)
		}
	}
}

func TestDeleteDeployment(t *testing.T) {
	tests := []struct {
		name           string
		deploymentName string
		namespace      string
		outErr         error
		skipCreate     bool
	}{
		{
			name:           "basic positive",
			deploymentName: "aDeployment",
			namespace:      "someNamespace",
			outErr:         nil,
		},
		{
			name:           "ignore not found",
			namespace:      "testNamespace",
			deploymentName: "iDoNotExist",
			outErr:         nil,
			skipCreate:     true,
		},
	}
	client := fake.NewClient()
	kc := NewWithClient(context.Background(), client)
	for _, test := range tests {
		if !test.skipCreate {
			cm, err := client.Kubernetes().AppsV1().Deployments(test.namespace).Create(&appsv1.Deployment{
				ObjectMeta: meta_v1.ObjectMeta{
					Name: test.deploymentName,
				}})
			if cm == nil {
				t.Errorf("No deployment created. Should have created %s", test.deploymentName)
			}
			if err != nil {
				t.Errorf("Error creating deployment %s : %v", test.deploymentName, err)
			}
			cm, err = client.Kubernetes().AppsV1().Deployments(test.namespace).Get(test.deploymentName, meta_v1.GetOptions{})
			if cm == nil {
				t.Errorf("Deployment %s not found", test.deploymentName)
			}
			if err != nil {
				t.Errorf("Error getting deployment %s : %v", test.deploymentName, err)
			}
		}
		err := kc.DeleteDeployment(test.deploymentName, test.namespace)
		if err != test.outErr {
			t.Errorf("Expected %v returned from DeleteDeployment, but got %v", test.outErr, err)
		}
		_, err = client.Kubernetes().AppsV1().Deployments(test.namespace).Get(test.deploymentName, meta_v1.GetOptions{})
		if !api_errors.IsNotFound(err) {
			t.Errorf("Deleted deployment %v still exists", test.deploymentName)
		}
	}
}

func TestDeleteNamespace(t *testing.T) {
	tests := []struct {
		name       string
		namespace  string
		outErr     error
		skipCreate bool
	}{
		{
			name:      "basic positive",
			namespace: "someNamespace",
			outErr:    nil,
		},
		{
			name:       "ignore not found",
			namespace:  "iDoNotExist",
			outErr:     nil,
			skipCreate: true,
		},
	}
	client := fake.NewClient()
	kc := NewWithClient(context.Background(), client)
	for _, test := range tests {
		cm, err := client.Kubernetes().CoreV1().Namespaces().Create(&v1.Namespace{
			ObjectMeta: meta_v1.ObjectMeta{
				Name: test.namespace,
			}})
		if !test.skipCreate {
			if cm == nil {
				t.Errorf("No namespace created. Should have created %s", test.namespace)
			}
			if err != nil {
				t.Errorf("Error creating namespace %s : %v", test.namespace, err)
			}
			cm, err = client.Kubernetes().CoreV1().Namespaces().Get(test.namespace, meta_v1.GetOptions{})
			if cm == nil {
				t.Errorf("Namespace %s not found", test.namespace)
			}
			if err != nil {
				t.Errorf("Error getting namespace %s : %v", test.namespace, err)
			}
		}
		err = kc.DeleteNamespace(test.namespace)
		if err != test.outErr {
			t.Errorf("Expected %v returned from DeleteNamespace, but got %v", test.outErr, err)
		}
		_, err = client.Kubernetes().CoreV1().Namespaces().Get(test.namespace, meta_v1.GetOptions{})
		if !api_errors.IsNotFound(err) {
			t.Errorf("Deleted namespace %v still exists", test.namespace)
		}
	}
}

func TestDeleteValidatingWebhookConfiguration(t *testing.T) {
	tests := []struct {
		name    string
		vwcName string
		outErr  error
	}{
		{
			name:    "basic positive",
			vwcName: "ConfigMyVWCPlz",
			outErr:  nil,
		},
	}
	client := fake.NewClient()
	kc := NewWithClient(context.Background(), client)
	for _, test := range tests {
		vwc, err := client.Kubernetes().AdmissionregistrationV1beta1().ValidatingWebhookConfigurations().Create(&v1beta1.ValidatingWebhookConfiguration{
			ObjectMeta: meta_v1.ObjectMeta{
				Name: test.vwcName,
			}})
		if vwc == nil {
			t.Errorf("No ValidatingWebhookConfiguration created. Should have created %s", test.vwcName)
		}
		if err != nil {
			t.Errorf("Error creating ValidatingWebhookConfiguration %s : %v", test.vwcName, err)
		}
		vwc, err = client.Kubernetes().AdmissionregistrationV1beta1().ValidatingWebhookConfigurations().Get(test.vwcName, meta_v1.GetOptions{})
		if vwc == nil {
			t.Errorf("ValidatinWebhookConfiguration %s not found", test.vwcName)
		}
		if err != nil {
			t.Errorf("Error getting ValidatingWebhookConfiguration %s : %v", test.vwcName, err)
		}
		err = kc.DeleteValidatingWebhookConfiguration(test.vwcName)
		if err != test.outErr {
			t.Errorf("Expected %v returned from DeleteValidatingWebhookConfiguration, but got %v", test.outErr, err)
		}
		_, err = client.Kubernetes().AdmissionregistrationV1beta1().ValidatingWebhookConfigurations().Get(test.vwcName, meta_v1.GetOptions{})
		if !api_errors.IsNotFound(err) {
			t.Errorf("Deleted ValidatingWebhookConfiguration %v still exists", test.vwcName)
		}
	}
}

func TestAddRemoveClusterAdmin(t *testing.T) {
	tests := []struct {
		name    string
		crbUser string
		outErr  error
	}{
		{
			name:    "basic positive",
			crbUser: "weBeSushi",
			outErr:  nil,
		},
	}
	client := fake.NewClient()
	kc := NewWithClient(context.Background(), client)
	for _, test := range tests {
		crbName := fmt.Sprintf("%v-cluster-admin-binding", test.crbUser)
		err := kc.AddClusterAdmin(test.crbUser)
		if err != nil {
			t.Errorf("Error while adding cluster admin: %v", err)
		}
		crb, err := client.Kubernetes().RbacV1().ClusterRoleBindings().Get(crbName, meta_v1.GetOptions{})
		if crb == nil {
			t.Errorf("No ClusterRoleBinding found created. Should have created %s", test.crbUser)
		}
		if err != nil {
			t.Errorf("Error getting ClusterRoleBinding %s : %v", crbName, err)
		}
		err = kc.RemoveClusterAdmin(test.crbUser)
		if err != test.outErr {
			t.Errorf("Expected %v returned from RemoveClusterAdmin, but got %v", test.outErr, err)
		}
		_, err = client.Kubernetes().RbacV1().ClusterRoleBindings().Get(test.crbUser, meta_v1.GetOptions{})
		if !api_errors.IsNotFound(err) {
			t.Errorf("Deleted ClusterRoleBinding %v still exists", crbName)
		}
	}
}
