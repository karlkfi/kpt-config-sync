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
  stackdriver/kubernetes:
    metric:
      prefix: "kubernetes.io/internal/addons/config_sync/"
      skip_create_descriptor: false
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
  filter/kubernetes:
    metrics:
      include:
        match_type: regexp
        metric_names:
          - kustomize.*
          - api_duration_seconds
          - reconciler_errors
          - pipeline_error_observed
          - reconcile_duration_seconds
          - parser_duration_seconds
          - declared_resources
          - apply_operations_total
          - apply_duration_seconds
          - resource_fights_total
          - remediate_duration_seconds
          - resource_conflicts_total
          - internal_errors_total
          - rendering_count_total
          - skip_rendering_count_total
          - resource_override_count_total
          - git_sync_depth_override_count_total
          - no_ssl_verify_count_total
  metricstransform/kubernetes:
    transforms:
      - include: declared_resources
        action: update
        new_name: current_declared_resources
      - include: reconciler_errors
        action: update
        new_name: last_reconciler_errors
      - include: pipeline_error_observed
        action: update
        new_name: last_pipeline_error_observed
      - include: apply_operations_total
        action: update
        new_name: apply_operations_count
      - include: resource_fights_total
        action: update
        new_name: resource_fights_count
      - include: resource_conflicts_total
        action: update
        new_name: resource_conflicts_count
      - include: internal_errors_total
        action: update
        new_name: internal_errors_count
      - include: rendering_count_total
        action: update
        new_name: rendering_count
      - include: skip_rendering_count_total
        action: update
        new_name: skip_rendering_count
      - include: resource_override_count_total
        action: update
        new_name: resource_override_count
      - include: git_sync_depth_override_count_total
        action: update
        new_name: git_sync_depth_override_count
      - include: no_ssl_verify_count_total
        action: update
        new_name: no_ssl_verify_count
extensions:
  health_check:
service:
  extensions: [health_check]
  pipelines:
    metrics/cloudmonitoring:
      receivers: [opencensus]
      processors: [batch, filter/cloudmonitoring]
      exporters: [stackdriver]
    metrics/prometheus:
      receivers: [opencensus]
      processors: [batch]
      exporters: [prometheus]
    metrics/kubernetes:
      receivers: [opencensus]
      processors: [batch, filter/kubernetes, metricstransform/kubernetes]
      exporters: [stackdriver/kubernetes]`
)
