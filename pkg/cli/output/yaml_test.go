package output

import (
	"bytes"
	"testing"

	policyhierarchy "github.com/google/stolos/pkg/api/policyhierarchy/v1"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestPrintForNamespace(t *testing.T) {
	tests := []struct {
		namespace string
		object    runtime.Object
		output    string
	}{
		{
			namespace: "default",
			object: &policyhierarchy.StolosResourceQuota{
				TypeMeta: meta.TypeMeta{
					Kind: "Namespace",
				},
				ObjectMeta: meta.ObjectMeta{
					Name:      "bunny",
					Namespace: "foofoo",
				},
			},
			output: `# Namespace: "default"
#
kind: Namespace
metadata:
  creationTimestamp: null
  name: bunny
  namespace: foofoo
spec:
  status: {}
`,
		},
	}

	for i, test := range tests {
		b := bytes.Buffer{}
		err := PrintForNamespace(test.namespace, test.object, &b)
		if err != nil {
			t.Errorf("[%v] Unexpected error: %v", i, err)

		}
		if b.String() != test.output {
			t.Errorf("[%v] Expected:\n%v\nactual:\n%v",
				i, test.output, b.String())
		}
	}
}
