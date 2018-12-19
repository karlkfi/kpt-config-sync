package parse

import (
	"github.com/google/nomos/pkg/api/policyhierarchy/v1"
	"github.com/google/nomos/pkg/api/policyhierarchy/v1alpha1"
	"github.com/google/nomos/pkg/client/restconfig"
	"github.com/google/nomos/pkg/policyimporter/filesystem"
	"github.com/pkg/errors"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/kubernetes/pkg/kubectl/genericclioptions"
)

// Ext implements filesystem.ParserExtension.
type Ext struct {
	VP    filesystem.VisitorProvider
	Syncs []*v1alpha1.Sync
}

// Visitors returns a filesystem.VisitorProvider that returns the visitors the parser should execute.
func (e *Ext) Visitors() filesystem.VisitorProvider { return e.VP }

// SyncResources returns additional resources that should be synced to the API server.
func (e *Ext) SyncResources() []*v1alpha1.Sync { return e.Syncs }

// Parse parses a GKE Policy Directory with a Parser using the specified Parser optional arguments.
// Exits early if it encounters parsing/validation errors.
func Parse(dir string, parserOpt filesystem.ParserOpt) (*v1.AllPolicies, error) {
	clientConfig, err := restconfig.NewClientConfig()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to get kubectl config")
	}

	restConfig, err := clientConfig.ClientConfig()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to get rest.Config")
	}

	// TODO(119066037): Override the host in a way that doesn't involve overwriting defaults set internally in client-go.
	clientcmd.ClusterDefaults = clientcmdapi.Cluster{Server: restConfig.Host}
	p, err := filesystem.NewParser(&genericclioptions.ConfigFlags{}, parserOpt)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create parser")
	}

	resources, err := p.Parse(dir)
	if err != nil {
		return nil, errors.Wrap(err, "Found issues")
	}

	return resources, nil
}
