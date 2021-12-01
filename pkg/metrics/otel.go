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
      prefix: "custom.googleapis.com/opencensus/config_sync/"
      skip_create_descriptor: true
    retry_on_failure:
      enabled: true
    sending_queue:
      enabled: true
processors:
  batch:
  # b/204120800: removing last_apply_timestamp from code is intractable since a lot of test code depends on it. Hence filtering it here
  filter/cloudmonitoring:
    metrics:
      exclude:
        match_type: strict
        metric_names:
          - last_apply_timestamp
  metricstransform:
    transforms:
    # These transforms are needed as part of fleet metrics mapping.
    # We would need to adjust testcases: ValidateDeclaredResources()
    # b/204120800
    #
    #  - include: declared_resources
    #    action: update
    #    new_name: current_declared_resources
    #  - include: reconciler_errors
    #    action: update
    #    new_name: last_reconciler_errors
      - include: .*
        match_type: regexp
        action: update
        operations:
          - action: add_label
            new_label: cluster
            new_value: {{.ClusterName}}
extensions:
  health_check:
service:
  extensions: [health_check]
  pipelines:
    metrics/cloudmonitoring:
      receivers: [opencensus]
      processors: [batch, filter/cloudmonitoring, metricstransform]
      exporters: [stackdriver]
    metrics/prometheus:
      receivers: [opencensus]
      processors: [batch, metricstransform]
      exporters: [prometheus]
`
)
