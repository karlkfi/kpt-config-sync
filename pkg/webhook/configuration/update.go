package configuration

import (
	"context"

	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/util/discovery"
	admissionv1 "k8s.io/api/admissionregistration/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Update modifies the ValidatingWebhookConfiguration on the cluster to match
// all types declared in objs.
//
// Returns an error if the API Server returns invalid API Resource lists or
// there is a problem updating the Configuration.
func Update(ctx context.Context, c client.Client, dc discovery.ServerResourcer, objs []ast.FileObject) status.MultiError {
	if len(objs) == 0 {
		// Nothing to do.
		return nil
	}

	_, _, err := dc.ServerGroupsAndResources()
	if err != nil {
		// Likely transient. If not, there's either a serious bug in our code or
		// something very wrong with the API Server.
		return status.APIServerError(err, "unable to list API Resources")
	}

	gvks := toGVKs(objs)
	newCfg := toWebhookConfiguration(gvks)
	if newCfg == nil {
		// The repository declares no objects, so there's nothing to do.
		return nil
	}

	oldCfg := &admissionv1.ValidatingWebhookConfiguration{}
	err = c.Get(ctx, client.ObjectKey{Name: Name}, oldCfg)
	switch {
	case apierrors.IsNotFound(err):
		// There is no Configuration yet, so nothing to merge.
		if err = c.Create(ctx, newCfg); err != nil {
			return status.APIServerError(err, "creating admission webhook")
		}
		return nil
	case err != nil:
		// Should be rare, but most likely will be a permission error.
		return status.APIServerError(err, "getting admission webhook from API Server")
	}

	// We aren't yet concerned with removing stale rules, so just merge the two
	// together.
	newCfg = Merge(oldCfg, newCfg)
	if err = c.Update(ctx, newCfg); err != nil {
		return status.APIServerError(err, "applying changes to admission webhook")
	}
	return nil
}

func toGVKs(objs []ast.FileObject) []schema.GroupVersionKind {
	seen := make(map[schema.GroupVersionKind]bool)
	var gvks []schema.GroupVersionKind
	for _, o := range objs {
		gvk := o.GroupVersionKind()
		if !seen[gvk] {
			seen[gvk] = true
			gvks = append(gvks, gvk)
		}
	}
	// The order of GVKs is not deterministic, but we're using it for
	// toWebhookConfiguration which does not require its input to be sorted.
	return gvks
}
