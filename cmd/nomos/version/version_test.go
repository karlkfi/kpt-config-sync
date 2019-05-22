package version

import (
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/rest"
)

func TestVersion(t *testing.T) {
	tests := []struct {
		name     string
		version  string
		objects  []runtime.Object
		expected []string
		contexts []string
		configs  map[string]*rest.Config
	}{
		{
			name:     "specify zero clusters",
			version:  "v1.2.3",
			contexts: []string{},
			configs:  nil,
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
				"NAME            COMPONENT       VERSION",
				"                <client>        v1.2.3",
				"",
			},
		},
		{
			name:    "not installed",
			version: "v2.3.4-rc.5",
			expected: []string{
				"NAME            COMPONENT           VERSION",
				"                <client>            v2.3.4-rc.5",
				"config          config-management   NOT INSTALLED",
				"",
			},
			configs: map[string]*rest.Config{"config": nil},
		},
		{
			name:    "installed something",
			version: "v3.4.5-rc.6",
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
				"NAME            COMPONENT           VERSION",
				"                <client>            v3.4.5-rc.6",
				"config          config-management   v1.2.3-rc.42",
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
			versionInternal(test.configs, &b, test.contexts)
			actuals := strings.Split(b.String(), "\n")
			if diff := cmp.Diff(test.expected, actuals); diff != "" {
				t.Errorf(diff)
			}
		})
	}
}
