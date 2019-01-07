package compile

import (
	"encoding/json"

	yaml "gopkg.in/yaml.v2"
	"k8s.io/apimachinery/pkg/runtime"
)

// Resource represents a Resource along with a path to where it would exist in the cluster if
// one were to represent the objects from the cluster on the filesystem.
type Resource struct {
	Path string
	Obj  runtime.RawExtension
}

// ToYAML converts the object to a yaml representation and trims the spec and status field if they
// are empty.
func (r *Resource) ToYAML() (string, error) {
	b, err := json.Marshal(r.Obj.Object)
	if err != nil {
		return "", err
	}

	jsonObj := map[string]interface{}{}
	if err2 := yaml.Unmarshal(b, jsonObj); err2 != nil {
		return "", err2
	}

	// Remove empty spec/status since the YAML/JSON don't handle this.
	for _, field := range []string{"spec", "status"} {
		if got, found := jsonObj[field]; found {
			if fieldVal, ok := got.(map[interface{}]interface{}); ok {
				if len(fieldVal) == 0 {
					delete(jsonObj, field)
				}
			}
		}
	}

	b, err = yaml.Marshal(jsonObj)
	if err != nil {
		return "", err
	}
	return string(b), nil
}
