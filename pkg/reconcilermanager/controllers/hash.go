package controllers

import (
	"hash/fnv"

	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/util/json"
)

func hash(allData interface{}) ([]byte, error) {
	data, err := json.Marshal(allData)
	if err != nil {
		return nil, errors.Errorf("failed to marshal ConfigMaps data, error: %v", err)
	}
	h := fnv.New128()
	if n, err := h.Write(data); n < len(data) {
		return nil, errors.Errorf("failed to write configmap data, error: %v", err)
	}
	return h.Sum(nil), nil
}
