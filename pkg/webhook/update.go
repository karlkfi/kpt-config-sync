package webhook

import (
	"context"

	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/util/discovery"
	admissionv1 "k8s.io/api/admissionregistration/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apiextensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// UpdateAdmissionWebhookConfiguration modifies the
// ValidatingWebhookConfiguration on the cluster to match all types declared in
// objs.
//
// Returns an error if the API Server returns invalid API Resource lists or
// there is a problem updating the Configuration.
func UpdateAdmissionWebhookConfiguration(ctx context.Context, c client.Client, dc discovery.ServerResourcer, objs []ast.FileObject) status.MultiError {
	if len(objs) == 0 {
		// Nothing to do.
		return nil
	}

	_, lists, err := dc.ServerGroupsAndResources()
	if err != nil {
		// Likely transient. If not, there's either a serious bug in our code or
		// something very wrong with the API Server.
		return status.APIServerError(err, "unable to list API Resources")
	}

	mapper, mErr := newKindResourceMapper(lists)
	if mErr != nil {
		return mErr
	}
	// The ordering of which version of CRDs is added first has no impact. We
	// don't allow two CRDs to exist with the same metadata.name, and thus there
	// cannot be two CRDs for the same GroupResource due to the requirements of
	// CRD naming.
	v1Beta1CRDs, mErr := getV1Beta1CRDs(objs)
	if mErr != nil {
		return mErr
	}
	mapper.addV1Beta1CRDs(v1Beta1CRDs)
	v1CRDs, mErr := getV1CRDs(objs)
	if mErr != nil {
		return mErr
	}
	mapper.addV1CRDs(v1CRDs)

	gvks := toGVKs(objs)
	newCfg, mErr := toWebhookConfiguration(mapper, gvks)
	if mErr != nil {
		// Either there's something wrong with the resources the API Server gave us
		// or there's an error in our parsing logic.
		//
		// There's also a small chance of transient errors if a user or process
		// deletes a CRD just after we've successfully parsed a repository, and so
		// the type isn't available on the API Server.
		return mErr
	}
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
	newCfg = MergeWebhookConfigurations(oldCfg, newCfg)
	if err = c.Update(ctx, newCfg); err != nil {
		return status.APIServerError(err, "applying changes to admission webhook")
	}
	return nil
}

func getV1CRDs(objs []ast.FileObject) ([]apiextensionsv1.CustomResourceDefinition, status.MultiError) {
	var crds []apiextensionsv1.CustomResourceDefinition
	var errs status.MultiError
	for _, o := range objs {
		if o.GroupVersionKind() != kinds.CustomResourceDefinitionV1() {
			continue
		}
		s, err := o.Structured()
		if err != nil {
			errs = status.Append(errs, err)
			continue
		}
		crds = append(crds, *s.(*apiextensionsv1.CustomResourceDefinition))
	}
	return crds, errs
}

func getV1Beta1CRDs(objs []ast.FileObject) ([]apiextensionsv1beta1.CustomResourceDefinition, status.MultiError) {
	var crds []apiextensionsv1beta1.CustomResourceDefinition
	var errs status.MultiError
	for _, o := range objs {
		if o.GroupVersionKind() != kinds.CustomResourceDefinitionV1Beta1() {
			continue
		}
		s, err := o.Structured()
		if err != nil {
			errs = status.Append(errs, err)
			continue
		}
		crds = append(crds, *s.(*apiextensionsv1beta1.CustomResourceDefinition))
	}
	return crds, errs
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
