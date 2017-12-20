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

package policy_node_controller

import (
	"github.com/google/stolos/pkg/client/meta"
)

type Controller struct {
}

func NewController(client meta.Interface, stopChan chan struct{}) *Controller {
	return &Controller{}
}

func (c *Controller) Run() {
	// TODO(frankf)
}

func (c *Controller) Stop() {
	// TODO(frankf)
}
