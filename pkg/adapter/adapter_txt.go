/*
Copyright 2017 The Kubernetes Authors.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Text version of adapter that loads a text based representation of a namespace hierarchy into memory
package adapter

import (
	"encoding/json"
	"fmt"
	"io/ioutil"

	"github.com/mdruskin/kubernetes-enterprise-control/pkg/api/policyhierarchy/v1"
)

func Load(filename string) ([]v1.PolicyNode, error) {
	config, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("Error reading file: %s", err)
	}
	var nodes []v1.PolicyNode
	err = json.Unmarshal(config, &nodes)
	if err != nil {
		return nil, fmt.Errorf("Error parsing file: %s", err)
	}
	return nodes, nil
}
