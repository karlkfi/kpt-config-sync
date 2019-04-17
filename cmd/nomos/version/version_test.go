package version

import (
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/rest"
)

func TestLegacyVersion(t *testing.T) {
	tests := []struct {
		name     string
		version  string
		expected string
	}{
		{
			name:     "basic",
			version:  "v1.2.3-rc.4",
			expected: "v1.2.3-rc.4\n",
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			clientVersion = func() string {
				return test.version
			}
			var b strings.Builder
			versionInternal(nil, &b, nil, false)
			if b.String() != test.expected {
				t.Errorf("version()=%q, want: %q", b.String(), test.expected)
			}
		})
	}
}

func TestVersion(t *testing.T) {
	tests := []struct {
		name        string
		version     string
		objects     []runtime.Object
		expected    []string
		clusters    []string
		allClusters bool
		configs     map[string]*rest.Config
	}{
		{
			name:        "not installed",
			allClusters: true,
			version:     "v2.3.4-rc.5",
			expected: []string{
				"NAME     COMPONENT           VERSION",
				"         <client>            v2.3.4-rc.5",
				"config   config-management   <not installed>",
				"",
			},
			configs: map[string]*rest.Config{"config": nil},
		},
		{
			name:        "installed something",
			allClusters: true,
			version:     "v3.4.5-rc.6",
			objects: []runtime.Object{
				&unstructured.Unstructured{
					Object: map[string]interface{}{
						"kind":       "ConfigManagement",
						"apiVersion": "addons.sigs.k8s.io/v1alpha1",
						"metadata": map[string]interface{}{
							"name":      "config-management",
							"namespace": "",
						},
						"status": map[string]interface{}{
							"configManagementVersion": "v1.2.3-rc.42",
						},
					},
				},
			},
			expected: []string{
				"NAME     COMPONENT           VERSION",
				"         <client>            v3.4.5-rc.6",
				"config   config-management   v1.2.3-rc.42",
				"",
			},
			configs: map[string]*rest.Config{"config": nil},
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			clientVersion = func() string {
				return test.version
			}
			dynamicClient = func(c *rest.Config) (dynamic.Interface, error) {
				return fake.NewSimpleDynamicClient(runtime.NewScheme(), test.objects...), nil
			}
			var b strings.Builder
			c := func(time.Duration) (map[string]*rest.Config, error) {
				return test.configs, nil
			}
			versionInternal(c, &b, test.clusters, test.allClusters)
			actuals := strings.Split(b.String(), "\n")
			if !cmp.Equal(actuals, test.expected) {
				t.Errorf("version()=\n%+v\n\nwant:\n%+v\n\ndiff=\n%v",
					b.String(), test.expected, cmp.Diff(test.expected, actuals))
			}
		})
	}
}
