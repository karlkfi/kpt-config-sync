// Copyright 2022 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package fake

import (
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/filesystem/cmpath"
	"github.com/google/nomos/pkg/kinds"
	appsv1 "k8s.io/api/apps/v1"
)

// DeploymentObject initializes a Deployment.
func DeploymentObject(opts ...core.MetaMutator) *appsv1.Deployment {
	result := &appsv1.Deployment{TypeMeta: ToTypeMeta(kinds.Deployment())}
	defaultMutate(result)
	mutate(result, opts...)

	return result
}

// Deployment returns a Deployment in a FileObject.
func Deployment(dir string, opts ...core.MetaMutator) ast.FileObject {
	relative := cmpath.RelativeSlash(dir).Join(cmpath.RelativeSlash("deployment.yaml"))
	return FileObject(DeploymentObject(opts...), relative.SlashPath())
}
