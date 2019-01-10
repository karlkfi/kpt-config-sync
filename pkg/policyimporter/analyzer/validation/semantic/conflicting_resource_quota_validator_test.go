package semantic

import (
	"testing"

	"github.com/google/nomos/pkg/api/policyhierarchy/v1alpha1"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/google/nomos/pkg/policyimporter/analyzer/validation/coverage"
	"github.com/google/nomos/pkg/policyimporter/analyzer/veterrors"
	"github.com/google/nomos/pkg/policyimporter/analyzer/veterrors/veterrorstest"
	"github.com/google/nomos/pkg/policyimporter/filesystem/nomospath"
	"github.com/google/nomos/pkg/util/multierror"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterregistry "k8s.io/cluster-registry/pkg/apis/clusterregistry/v1alpha1"
)

func TestConflictingResourceQuotaValidator(t *testing.T) {
	tests := []struct {
		name      string
		input     []ast.FileObject
		clusters  []clusterregistry.Cluster
		selectors []v1alpha1.ClusterSelector
		errors    []string
	}{
		{
			name: "Basic OK",
			input: []ast.FileObject{
				rq(metav1.ObjectMeta{Name: "foo"}, "dir1/src1"),
				rq(metav1.ObjectMeta{Name: "foo"}, "dir2/src2"),
			},
		},
		{
			name: "Two quotas in the same directory result in an 1008 error",
			input: []ast.FileObject{
				rq(metav1.ObjectMeta{Name: "foo"}, "dir1/src1"),
				rq(metav1.ObjectMeta{Name: "bar"}, "dir1/src2"),
			},
			errors: []string{veterrors.ConflictingResourceQuotaErrorCode},
		},
		{
			name: "Two quotas in the same directory targeted to different clusters are OK",
			input: []ast.FileObject{
				rq(metav1.ObjectMeta{
					Name: "foo",
					Annotations: map[string]string{
						v1alpha1.ClusterSelectorAnnotationKey: "sel-1",
					},
				}, "dir1/src1"),
				rq(metav1.ObjectMeta{
					Name: "bar",
					Annotations: map[string]string{
						v1alpha1.ClusterSelectorAnnotationKey: "sel-2",
					},
				}, "dir1/src2"),
			},
			clusters: []clusterregistry.Cluster{
				cluster("cluster-1", map[string]string{
					"env": "prod",
				}),
				cluster("cluster-2", map[string]string{
					"env": "test",
				}),
			},
			selectors: []v1alpha1.ClusterSelector{
				selector("sel-1", map[string]string{
					"env": "prod",
				}),
				selector("sel-2", map[string]string{
					"env": "test",
				}),
			},
		},
		{
			name: "One targeted quota and one general quota are not OK",
			input: []ast.FileObject{
				rq(metav1.ObjectMeta{
					Name: "foo",
					Annotations: map[string]string{
						v1alpha1.ClusterSelectorAnnotationKey: "sel-1",
					},
				}, "dir1/src1"),
				rq(metav1.ObjectMeta{
					Name: "bar",
				}, "dir1/src2"),
			},
			clusters: []clusterregistry.Cluster{
				cluster("cluster-1", map[string]string{
					"env": "prod",
				}),
			},
			selectors: []v1alpha1.ClusterSelector{
				selector("sel-1", map[string]string{
					"env": "prod",
				}),
			},
			errors: []string{veterrors.ConflictingResourceQuotaErrorCode},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var m multierror.Builder
			cov := coverage.NewForCluster(test.clusters, test.selectors, &m)

			v := NewConflictingResourceQuotaValidator(test.input, cov)
			v.Validate(&m)

			veterrorstest.ExpectErrors(test.errors, m.Build(), t)
		})
	}
}

func rq(meta metav1.ObjectMeta, dir string) ast.FileObject {
	q := corev1.ResourceQuota{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ResourceQuota",
		},
		ObjectMeta: meta,
	}
	return ast.NewFileObject(&q, nomospath.NewFakeRelative(dir))
}

func cluster(name string, labels map[string]string) clusterregistry.Cluster {
	return clusterregistry.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: labels,
		},
	}
}

func selector(name string, matchLabels map[string]string) v1alpha1.ClusterSelector {
	return v1alpha1.ClusterSelector{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: v1alpha1.ClusterSelectorSpec{
			Selector: metav1.LabelSelector{
				MatchLabels: matchLabels,
			},
		},
	}

}
