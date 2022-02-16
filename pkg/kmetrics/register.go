package kmetrics

import (
	"os"

	"contrib.go.opencensus.io/exporter/ocagent"
	"go.opencensus.io/stats/view"
)

const (
	kmResourceNamespaceName = "kmetrics-system"
	kmResourcePodName       = "kmetrics-manager"
)

// RegisterOCAgentExporter creates the OC Agent metrics exporter.
func RegisterOCAgentExporter() (*ocagent.Exporter, error) {
	err := os.Setenv(
		"OC_RESOURCE_LABELS",
		"k8s.namespace.name=\""+kmResourceNamespaceName+"\",k8s.pod.name=\""+kmResourcePodName+"\"")
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
