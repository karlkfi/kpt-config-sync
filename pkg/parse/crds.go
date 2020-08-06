package parse

import (
	"context"

	"github.com/google/nomos/pkg/importer/filesystem"
	"github.com/google/nomos/pkg/status"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func listCrds(ctx context.Context, reader client.Reader) filesystem.GetSyncedCRDs {
	return func() ([]*v1beta1.CustomResourceDefinition, status.MultiError) {
		list := v1beta1.CustomResourceDefinitionList{}
		err := reader.List(ctx, &list)
		if err != nil {
			return nil, status.APIServerError(err, "listing CRDs")
		}

		result := make([]*v1beta1.CustomResourceDefinition, len(list.Items))
		for i := range list.Items {
			result[i] = &list.Items[i]
		}
		return result, nil

	}
}
