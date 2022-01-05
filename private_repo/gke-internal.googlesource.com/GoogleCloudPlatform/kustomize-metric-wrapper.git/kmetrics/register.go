package kmetrics

import (
	"os"

	"contrib.go.opencensus.io/exporter/ocagent"
	"go.opencensus.io/stats/view"
)

const (
	km_resource_namespace_name = "kmetrics-system"
	km_resource_pod_name       = "kmetrics-manager"
)

// RegisterOCAgentExporter creates the OC Agent metrics exporter.
func RegisterOCAgentExporter() (*ocagent.Exporter, error) {
	err := os.Setenv(
		"OC_RESOURCE_LABELS",
		"k8s.namespace.name=\""+km_resource_namespace_name+"\",k8s.pod.name=\""+km_resource_pod_name+"\"")
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
