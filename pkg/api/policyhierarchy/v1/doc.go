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

// +k8s:deepcopy-gen=package,register
// +k8s:openapi-gen=true

// +groupName=k8us.k8s.io

// Package v1 contains the version 1 data definition for the PolicyNode custom
// resource.
//
// To regenerate clientset and deepcopy (why aren't these done in
// one tool?) run:
//
//     tools/generate-clientset.sh
package v1
