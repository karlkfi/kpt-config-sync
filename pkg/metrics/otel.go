package metrics

const (
	// OpenTelemetry is the app label for all otel resources.
	OpenTelemetry = "opentelemetry"

	// OtelAgentName is the name of the OpenTelemetry Agent.
	OtelAgentName = "otel-agent"

	// OtelCollectorName is the name of the OpenTelemetry Collector.
	OtelCollectorName = "otel-collector"

	// OtelCollectorStackdriver is the name of the Stackdriver OpenTelemetry Collector ConfigMap.
	OtelCollectorStackdriver = "otel-collector-stackdriver"

	// OtelCollectorCustomCM is the name of the custom OpenTelemetry Collector ConfigMap.
	OtelCollectorCustomCM = "otel-collector-custom"

	// MonitoringNamespace is the Namespace used for OpenTelemetry Collector deployment.
	MonitoringNamespace = "config-management-monitoring"

	// CollectorConfigStackdriver is the OpenTelemetry Collector configuration with
	// the Stackdriver exporter.
	CollectorConfigStackdriver = `receivers:
  opencensus:
exporters:
  prometheus:
    endpoint: :8675
    namespace: config_sync
  stackdriver:
    metric:
      prefix: config_sync
    retry_on_failure:
      enabled: true
    sending_queue:
      enabled: true
processors:
  batch:
extensions:
  health_check:
service:
  extensions: [health_check]
  pipelines:
    metrics:
      receivers: [opencensus]
      processors: [batch]
      exporters: [prometheus, stackdriver]
`
)
