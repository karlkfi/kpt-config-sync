package util

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/golang/glog"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// UnmarshalStatus populates the given struct from the given unstructured data.
func UnmarshalStatus(obj unstructured.Unstructured, status interface{}) error {
	statusRaw, found, err := unstructured.NestedFieldNoCopy(obj.Object, "status")
	if err != nil {
		return err
	}
	if !found || statusRaw == nil {
		return nil
	}
	statusJSON, err := json.Marshal(statusRaw)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(statusJSON, status); err != nil {
		return err
	}
	return nil
}

// jsonify marshals the given string array as a JSON string.
func jsonify(strs []string) string {
	errJSON, err := json.Marshal(strs)
	if err == nil {
		return string(errJSON)
	}

	// This code is not intended to be reached. It just provides a sane fallback
	// if there is ever an error from json.Marshal().
	glog.Errorf("Failed to JSONify strings: %v", err)
	var b strings.Builder
	b.WriteString("[")
	for i, statusErr := range strs {
		if i > 0 {
			b.WriteString(",")
		}
		b.WriteString(fmt.Sprintf("%q", statusErr))
	}
	b.WriteString("]")
	return b.String()
}
