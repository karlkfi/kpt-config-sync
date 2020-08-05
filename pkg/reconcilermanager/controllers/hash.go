package controllers

import (
	"hash/fnv"

	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/util/json"
)

func hash(allData map[string]map[string]string) ([]byte, error) {
	data, err := json.Marshal(allData)
	if err != nil {
		return nil, errors.Errorf("failed to marshal ConfigMaps data, error: %v", err)
	}
	h := fnv.New128()
	return h.Sum(data), nil
}
