package installer

import (
	"context"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/nomos/pkg/api/policyhierarchy/v1"
	"github.com/google/nomos/pkg/client/meta/fake"
	"github.com/google/nomos/pkg/installer/config"
	"github.com/google/nomos/pkg/process/kubectl"

	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestImporterConfigMap(t *testing.T) {
	testCases := []struct {
		name   string
		config config.Config
		want   []string
	}{
		{
			name: "git ssh",
			config: config.Config{
				Git: config.GitConfig{
					SyncRepo:           "git@github.com:user/foo-corp.git",
					UseSSH:             true,
					PrivateKeyFilename: "/some/path/id_rsa",
					KnownHostsFilename: "/some/path/known_hosts",
					SyncBranch:         "master",
					RootPolicyDir:      "foo-corp",
					SyncWaitSeconds:    60,
				},
			},
			want: []string{
				"GIT_SYNC_SSH=true",
				"GIT_SYNC_REPO=git@github.com:user/foo-corp.git",
				"GIT_SYNC_BRANCH=master",
				"GIT_SYNC_WAIT=60",
				"GIT_KNOWN_HOSTS=true",
				"GIT_COOKIE_FILE=false",
				"POLICY_DIR=foo-corp",
			},
		},
		{
			name: "git ssh, no known hosts",
			config: config.Config{
				Git: config.GitConfig{
					SyncRepo:           "git@github.com:user/foo-corp.git",
					UseSSH:             true,
					PrivateKeyFilename: "/some/path/id_rsa",
					SyncBranch:         "master",
					RootPolicyDir:      "foo-corp",
					SyncWaitSeconds:    60,
				},
			},
			want: []string{
				"GIT_SYNC_SSH=true",
				"GIT_SYNC_REPO=git@github.com:user/foo-corp.git",
				"GIT_SYNC_BRANCH=master",
				"GIT_SYNC_WAIT=60",
				"GIT_KNOWN_HOSTS=false",
				"GIT_COOKIE_FILE=false",
				"POLICY_DIR=foo-corp",
			},
		},
		{
			name: "git https",
			config: config.Config{
				Git: config.GitConfig{
					SyncRepo:        "https://github.com/sbochins-k8s/foo-corp-example.git",
					UseSSH:          false,
					SyncBranch:      "master",
					RootPolicyDir:   "foo-corp",
					SyncWaitSeconds: 60,
					CookieFilename:  "~/.gitcookies",
				},
			},
			want: []string{
				"GIT_SYNC_SSH=false",
				"GIT_SYNC_REPO=https://github.com/sbochins-k8s/foo-corp-example.git",
				"GIT_SYNC_BRANCH=master",
				"GIT_SYNC_WAIT=60",
				"GIT_KNOWN_HOSTS=false",
				"GIT_COOKIE_FILE=true",
				"POLICY_DIR=foo-corp",
			},
		},
		{
			name: "gcp",
			config: config.Config{
				GCP: config.GCPConfig{
					OrgID:              "1234",
					PrivateKeyFilename: "/some/file",
				},
			},
			want: []string{
				"ORG_ID=1234",
			},
		},
		{
			name: "gcp with api address",
			config: config.Config{
				GCP: config.GCPConfig{
					OrgID:              "1234",
					PrivateKeyFilename: "/some/file",
					PolicyAPIAddress:   "localhost:1234",
				},
			},
			want: []string{
				"ORG_ID=1234",
				"POLICY_API_ADDRESS=localhost:1234",
			},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			i := &Installer{c: tt.config}
			var got []string
			if strings.Contains(tt.name, "git") {
				got = i.gitConfigMapContent()
			} else if strings.Contains(tt.name, "gcp") {
				got = i.gcpConfigMapContent()
			} else {
				t.Errorf("test case name must contain either git or gcp")
			}
			if diff := cmp.Diff(got, tt.want); diff != "" {
				t.Errorf("expected %v got %v\ndiff %v", tt.want, got, diff)
			}
		})
	}
}

func TestSyncerConfigMap(t *testing.T) {
	testCases := []struct {
		name   string
		config config.Config
		want   []string
	}{
		{
			name: "git ssh, no known hosts",
			config: config.Config{
				Git: config.GitConfig{
					SyncRepo: "git@github.com:user/foo-corp.git",
				},
			},
			want: []string{
				"gcp.mode=false",
			},
		},
		{
			name: "gcp",
			config: config.Config{
				GCP: config.GCPConfig{
					OrgID:              "1234",
					PrivateKeyFilename: "/some/file",
				},
			},
			want: []string{
				"gcp.mode=true",
			},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			i := &Installer{c: tt.config}
			got := i.syncerConfigMapContent()
			if diff := cmp.Diff(got, tt.want); diff != "" {
				t.Errorf("expected %v got %v\ndiff %v", tt.want, got, diff)
			}
		})
	}
}

func TestInstaller_DeleteClusterPolicies(t *testing.T) {
	const p1name = "mostExcellentPolicy"
	const p2name = "mostHeinousPolicy"

	var err error
	client := fake.NewClient()
	_, err = client.PolicyHierarchy().NomosV1().ClusterPolicies().Create(&v1.ClusterPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name: p1name,
		}})
	if err != nil {
		t.Errorf("unexpected error: %s", err)
	}
	_, err = client.PolicyHierarchy().NomosV1().ClusterPolicies().Create(&v1.ClusterPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name: p2name,
		}})
	if err != nil {
		t.Errorf("unexpected error: %s", err)
	}

	cp, err := client.PolicyHierarchy().NomosV1().ClusterPolicies().List(metav1.ListOptions{
		IncludeUninitialized: true,
	})
	if err != nil {
		t.Errorf("error listing cluster policies: %v", err)
	}

	items := cp.Items
	if len(items) != 2 || items[0].Name != p1name || items[1].Name != p2name {
		t.Errorf("unexpected cluster policies list. "+
			"Wanted [ %s , %s ], got: %v", p1name, p2name, items)
	}

	i := &Installer{k: kubectl.NewWithClient(context.Background(), client)}
	err = i.DeleteClusterPolicies()
	if err != nil {
		t.Error(err)
	}

	cp, err = client.PolicyHierarchy().NomosV1().ClusterPolicies().List(metav1.ListOptions{
		IncludeUninitialized: true,
	})
	if err != nil {
		t.Errorf("error listing cluster policies: %v", err)
	}
	if len(cp.Items) != 0 {
		t.Errorf("expected empty list but got %v", items)
	}
}

func TestInstaller_DeletePolicyNodes(t *testing.T) {
	const n1name = "billNode"
	const n2name = "tedNode"

	var err error
	client := fake.NewClient()

	_, err = client.PolicyHierarchy().NomosV1().PolicyNodes().Create(&v1.PolicyNode{
		ObjectMeta: metav1.ObjectMeta{
			Name: n1name,
		}})
	if err != nil {
		t.Errorf("unexpected error: %s", err)
	}
	_, err = client.PolicyHierarchy().NomosV1().PolicyNodes().Create(&v1.PolicyNode{
		ObjectMeta: metav1.ObjectMeta{
			Name: n2name,
		}})
	if err != nil {
		t.Errorf("unexpected error: %s", err)
	}

	pn, err := client.PolicyHierarchy().NomosV1().PolicyNodes().List(metav1.ListOptions{
		IncludeUninitialized: true,
	})
	if err != nil {
		t.Errorf("error listing policy nodes: %v", err)
	}

	items := pn.Items
	if len(items) != 2 || items[0].Name != n1name || items[1].Name != n2name {
		t.Errorf("unexpected policy nodes list. "+
			"Wanted [ %s, %s], got: %v", n1name, n2name, items)
	}

	i := &Installer{k: kubectl.NewWithClient(context.Background(), client)}
	err = i.DeletePolicyNodes()
	if err != nil {
		t.Error(err)
	}

	pn, err = client.PolicyHierarchy().NomosV1().PolicyNodes().List(metav1.ListOptions{})
	if err != nil {
		t.Errorf("error listing policy nodes: %v", err)
	}
	if len(pn.Items) != 0 {
		t.Errorf("expected empty list but got %v", items)
	}
}

func TestInstaller_DeleteDeprecatedCRDs(t *testing.T) {
	client := fake.NewClient()

	_, err := client.APIExtensions().ApiextensionsV1beta1().CustomResourceDefinitions().Create(&v1beta1.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name: "namespaceselectors.nomos.dev",
		},
		Spec: v1beta1.CustomResourceDefinitionSpec{
			Group:   "nomos.dev",
			Version: "v1",
		},
	})
	if err != nil {
		t.Error(err)
	}

	i := &Installer{k: kubectl.NewWithClient(context.Background(), client)}

	i.deleteDeprecatedCRDs()

	_, err = client.APIExtensions().ApiextensionsV1beta1().CustomResourceDefinitions().Get("namespaceselectors.nomos.dev", metav1.GetOptions{})

	if !errors.IsNotFound(err) {
		t.Errorf("Expected the deprecated CRD to be deleted, but it was not. Got err=%s", err)
	}
}

func TestInstaller_InstallNomos(t *testing.T) {
	client := fake.NewClient()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	i := &Installer{
		k: kubectl.NewWithClient(ctx, client),
		c: config.Config{
			Clusters: []config.Cluster{
				{
					Name:    "cluster-name",
					Context: "cluster-context",
				},
			},
		},
	}
	if err := i.installNomos("cluster-context"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	n, err := i.k.GetClusterName()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n != "cluster-name" {
		t.Errorf("ClusterName==%v, want: %v", n, "cluster-name")
	}
}

func TestInstaller_UninstallNomos(t *testing.T) {
	client := fake.NewClient()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	i := &Installer{
		k: kubectl.NewWithClient(ctx, client),
		c: config.Config{
			Clusters: []config.Cluster{
				{
					Name:    "cluster-name",
					Context: "cluster-context",
				},
			},
		},
	}

	if err := i.k.CreateClusterName("cluster-name"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := i.deleteNomos(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := i.k.GetClusterName(); err != nil {
		if !errors.IsNotFound(err) {
			t.Fatalf("unexpected error: %v", err)
		}
	} else {
		t.Fatalf("expected error, got nil")
	}
}
