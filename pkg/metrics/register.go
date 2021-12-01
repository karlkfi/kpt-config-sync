package metrics

import (
	"fmt"
	"os"
	"regexp"

	"contrib.go.opencensus.io/exporter/ocagent"
	"go.opencensus.io/stats/view"
)

// RegisterOCAgentExporter creates the OC Agent metrics exporter.
func RegisterOCAgentExporter() (*ocagent.Exporter, error) {
	// Update OC_RESOURCE_LABELS defined in go.opencensus.io/resource/resource.go
	// So that each OC agent will have corresponding resource labels
	// Adding pod name and namespace name can have metrics identified as container_pod
	// Cluster name & cluster location & project name are attached automatically
	podName, namespace := GetResourceLabels()
	err := os.Setenv("OC_RESOURCE_LABELS", fmt.Sprintf("k8s.namespace.name=%q,k8s.pod.name=%q", namespace, podName))
	if err != nil {
		return nil, err
	}
	oce, err := ocagent.NewExporter(
		ocagent.WithInsecure(),
	)
	if err != nil {
		return nil, err
	}

	view.RegisterExporter(oce)
	return oce, nil
}

// GetResourceLabels gets the namespace and filters the reconciler name from the pod name that are exposed via the Downward API
// (https://kubernetes.io/docs/tasks/inject-data-application/environment-variable-expose-pod-information/#the-downward-api).
// If the regex filter fails, the entire pod name is returned.
func GetResourceLabels() (string, string) {
	podName := os.Getenv("RECONCILER_NAME")
	namespace := os.Getenv("NAMESPACE_NAME")
	regex := regexp.MustCompile(`(?:([a-z0-9]+(?:-[a-z0-9]+)*))-[a-z0-9]+-(?:[a-z0-9]+)`)
	ss := regex.FindStringSubmatch(podName)
	if ss != nil {
		return ss[1], namespace
	}
	return podName, namespace
}

// RegisterReconcilerManagerMetricsViews registers the views so that recorded metrics can be exported in the reconciler manager.
func RegisterReconcilerManagerMetricsViews() error {
	return view.Register(ReconcileDurationView)
}

// RegisterReconcilerMetricsViews registers the views so that recorded metrics can be exported in the reconcilers.
func RegisterReconcilerMetricsViews() error {
	return view.Register(
		APICallDurationView,
		ReconcilerErrorsView,
		ParserDurationView,
		LastApplyTimestampView,
		LastSyncTimestampView,
		DeclaredResourcesView,
		ApplyOperationsView,
		ApplyDurationView,
		ResourceFightsView,
		RemediateDurationView,
		ResourceConflictsView,
		InternalErrorsView,
		RenderingCountView,
		SkipRenderingCountView,
		ResourceOverrideCountView,
		GitSyncDepthOverrideCountView,
		NoSSLVerifyCountView,
		PipelineErrorView,
	)
}
