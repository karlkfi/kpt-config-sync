package coverage

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	v1 "github.com/google/nomos/pkg/api/policyhierarchy/v1"
	"github.com/google/nomos/pkg/policyimporter/analyzer/vet"
	"github.com/google/nomos/pkg/util/multierror"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterregistry "k8s.io/cluster-registry/pkg/apis/clusterregistry/v1alpha1"
)

func TestValidateObject(t *testing.T) {
	tests := []struct {
		name      string
		clusters  []clusterregistry.Cluster
		selectors []v1.ClusterSelector
		errors    []vet.Error
		objects   []metav1.Object
	}{
		{
			name:      "basic",
			clusters:  []clusterregistry.Cluster{},
			selectors: []v1.ClusterSelector{},
		},
		{
			name: "the two clusters",
			clusters: []clusterregistry.Cluster{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "cluster-1",
						Labels: map[string]string{
							"env": "prod",
						},
					},
					Spec: clusterregistry.ClusterSpec{},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "cluster-2",
						Labels: map[string]string{
							"env": "prod",
						},
					},
					Spec: clusterregistry.ClusterSpec{},
				},
			},
			selectors: []v1.ClusterSelector{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "sel-1",
					},
					Spec: v1.ClusterSelectorSpec{
						Selector: metav1.LabelSelector{
							MatchLabels: map[string]string{
								"env": "prod",
							},
						},
					},
				},
			},
			objects: []metav1.Object{
				&metav1.ObjectMeta{
					Name: "object",
					Annotations: map[string]string{
						v1.ClusterSelectorAnnotationKey: "sel-1",
					},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var b multierror.Builder
			cov := NewForCluster(test.clusters, test.selectors, &b)
			for _, o := range test.objects {
				cov.ValidateObject(o, &b)
			}
			rawE := b.Build()
			var actual []vet.Error
			if rawE != nil {
				actual = rawE.(multierror.MultiError).Errors()
			}
			if !cmp.Equal(actual, test.errors) {
				t.Errorf("cov.Errors()=%v, want:\n%v,diff:\n%v",
					actual, test.errors, cmp.Diff(actual, test.errors))
			}
		})
	}
}

func TestMapToClusters(t *testing.T) {
	tests := []struct {
		name      string
		clusters  []clusterregistry.Cluster
		selectors []v1.ClusterSelector
		object    metav1.Object
		expected  []string
	}{
		{
			name:      "basic",
			clusters:  []clusterregistry.Cluster{},
			selectors: []v1.ClusterSelector{},
			object:    &metav1.ObjectMeta{},
			expected:  []string{""},
		},
		{
			name: "two clusters",
			clusters: []clusterregistry.Cluster{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "cluster-1",
						Labels: map[string]string{
							"env": "prod",
						},
					},
					Spec: clusterregistry.ClusterSpec{},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "cluster-2",
						Labels: map[string]string{
							"env": "prod",
						},
					},
					Spec: clusterregistry.ClusterSpec{},
				},
			},
			selectors: []v1.ClusterSelector{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "sel-1",
					},
					Spec: v1.ClusterSelectorSpec{
						Selector: metav1.LabelSelector{
							MatchLabels: map[string]string{
								"env": "prod",
							},
						},
					},
				},
			},
			object: &metav1.ObjectMeta{
				Name: "object",
				Annotations: map[string]string{
					v1.ClusterSelectorAnnotationKey: "sel-1",
				},
			},
			expected: []string{"cluster-1", "cluster-2"},
		},
		{
			name: "mismatching clusters",
			clusters: []clusterregistry.Cluster{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "cluster-1",
						Labels: map[string]string{
							"env": "test",
						},
					},
					Spec: clusterregistry.ClusterSpec{},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "cluster-2",
						Labels: map[string]string{
							"env": "prod",
						},
					},
					Spec: clusterregistry.ClusterSpec{},
				},
			},
			selectors: []v1.ClusterSelector{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "sel-1",
					},
					Spec: v1.ClusterSelectorSpec{
						Selector: metav1.LabelSelector{
							MatchLabels: map[string]string{
								"env": "prod",
							},
						},
					},
				},
			},
			object: &metav1.ObjectMeta{
				Name: "object",
				Annotations: map[string]string{
					v1.ClusterSelectorAnnotationKey: "sel-1",
				},
			},
			expected: []string{"cluster-2"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var b multierror.Builder
			cov := NewForCluster(test.clusters, test.selectors, &b)
			if b.Build() != nil {
				t.Fatalf("unexpected error: %v", b.Build())
			}
			actual := cov.MapToClusters(test.object)
			if !cmp.Equal(test.expected, actual) {
				t.Errorf("MapToClusters()=%v, want:\n%v,diff:\n%v",
					actual, test.expected, cmp.Diff(test.expected, actual))
			}
		})
	}
}
