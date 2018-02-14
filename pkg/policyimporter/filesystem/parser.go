/*
Copyright 2017 The Stolos Authors.
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

package filesystem

import (
	"github.com/golang/glog"
	cmdutil "k8s.io/kubernetes/pkg/kubectl/cmd/util"
	"k8s.io/kubernetes/pkg/kubectl/resource"
	"k8s.io/kubernetes/pkg/kubectl/validation"
)

type parser struct {
	factory   cmdutil.Factory
	schema    validation.Schema
	inCluster bool
}

// NewParser creates a new parser that reads files on disk and builds Stolos CRDs.
// inCluster boolean determines if this is running in a cluster and can talk to api server.
func NewParser(inCluster bool) *parser {
	p := parser{
		factory:   cmdutil.NewFactory(nil),
		inCluster: inCluster,
	}

	// If running in cluster, validate objects using OpenAPI schema downloaded from the API server.
	// TODO(frankf): Bake in the swagger spec when running outside the cluster?
	if inCluster {
		schema, err := p.factory.Validator(true)
		if err != nil {
			glog.Fatalf("Failed to get schema: %v", err)
		}
		p.schema = schema
	}

	return &p
}

func (p *parser) Parse(policyDir string) {
	b := p.factory.NewBuilder().Internal()

	if p.inCluster {
		b = b.Schema(p.schema)
	} else {
		b = b.Local()
	}

	result := b.
		ContinueOnError().
		FilenameParam(false, &resource.FilenameOptions{Recursive: true, Filenames: []string{policyDir}}).
		Do()

	infos, err := result.Infos()
	if err != nil {
		glog.Fatalf("Failed to read resources from %s: %p", policyDir, err)
	}

	for _, i := range infos {
		glog.Infof("%s -> %+v", i.Source, i.AsVersioned())
	}

	// TODO(frankf): Validate objects and build CRDs.
	// TODO(frankf): Add tests for valid/invalid yaml/json hierarchy.
}
