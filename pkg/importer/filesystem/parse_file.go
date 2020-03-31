package filesystem

import (
	"io/ioutil"
	"path/filepath"
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/kubernetes/scheme"
)

func parseFile(path string) ([]*unstructured.Unstructured, error) {
	switch filepath.Ext(path) {
	case ".yml", ".yaml":
		contents, err := ioutil.ReadFile(path)
		if err != nil {
			return nil, err
		}
		return parseYAMLFile(contents)
	case ".json":
		contents, err := ioutil.ReadFile(path)
		if err != nil {
			return nil, err
		}
		return parseJSONFile(contents)
	default:
		return nil, nil
	}
}

func isEmptyYAMLDocument(document string) bool {
	lines := strings.Split(document, "\n")
	for _, line := range lines {
		if len(strings.TrimSpace(line)) == 0 {
			// Ignore empty/whitespace-only lines.
			continue
		}
		if strings.HasPrefix(strings.TrimSpace(line), "#") {
			// Ignore comment lines.
			continue
		}
		return false
	}
	return true
}

func parseYAMLFile(contents []byte) ([]*unstructured.Unstructured, error) {
	// We have to manually split documents with the YAML separator since by default
	// yaml.Unmarshal only unmarshalls the first document, but a file may contain multiple.
	var result []*unstructured.Unstructured

	// A newline followed by triple-dash begins a new YAML document, so this is safe.
	documents := strings.Split(string(contents), "\n---")
	for _, document := range documents {
		if isEmptyYAMLDocument(document) {
			// Kubernetes ignores empty documents.
			continue
		}

		var u unstructured.Unstructured
		_, _, err := scheme.Codecs.UniversalDeserializer().Decode([]byte(document), nil, &u)
		if err != nil {
			return nil, err
		}
		result = append(result, &u)
	}

	return result, nil
}

func parseJSONFile(contents []byte) ([]*unstructured.Unstructured, error) {
	if len(contents) == 0 {
		// While an empty files is not valid JSON, Kubernetes allows empty JSON
		// files when applying multiple files.
		return nil, nil
	}
	// Kubernetes does not recognize arrays of Kubernetes objects in JSON files.
	// A single file must contain exactly one Kubernetes object, so we don't
	// have to do the same work we had to do for YAML.
	var u unstructured.Unstructured
	err := u.UnmarshalJSON(contents)
	return []*unstructured.Unstructured{&u}, err
}
