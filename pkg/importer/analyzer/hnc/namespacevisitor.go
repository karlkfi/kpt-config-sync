// Package hnc adds additional HNC-understandable annotation and labels to namespaces managed by
// ACM. Please send code reviews to gke-kubernetes-hnc-core@.
package hnc

import "github.com/google/nomos/pkg/constants"

const (
	// AnnotationKeyV1A2 is the annotation that indicates the namespace hierarchy is
	// not managed by the Hierarchical Namespace Controller (http://bit.ly/k8s-hnc-design) but
	// someone else, "configmanagement.gke.io" in this case.
	AnnotationKeyV1A2 = "hnc.x-k8s.io/managed-by"

	// OriginalHNCManagedByValue is the annotation that stores the original value of the
	// hnc.x-k8s.io/managed-by annotation before Config Sync overrides the annotation.
	OriginalHNCManagedByValue = constants.ConfigSyncPrefix + "original-hnc-managed-by-value"

	// DepthSuffix is a label suffix for hierarchical namespace depth.
	// See definition at http://bit.ly/k8s-hnc-design#heading=h.1wg2oqxxn6ka.
	DepthSuffix = ".tree.hnc.x-k8s.io/depth"
)
