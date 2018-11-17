# Monitoring and Debugging

## Logging

GKE Policy Management follows
[K8S logging convention](https://github.com/kubernetes/community/blob/master/contributors/devel/logging.md).
By default, all binaries log at V(2).

List all nomos-system pods:

```console
$ kubectl get deployment -n nomos-system
NAME                                 DESIRED   CURRENT   UP-TO-DATE   AVAILABLE   AGE
git-policy-importer                  1         1         1            1           13d
resourcequota-admission-controller   1         1         1            1           9d
syncer                               1         1         1            1           13d
```

To see logs for pod:

```console
$ kubectl logs -l app=syncer --namespace nomos-system
```

git-policy-importer pod has two containers.

To see logs for policy-importer container:

```console
$ kubectl logs -l app=git-policy-importer -c policy-importer -n nomos-system
```

To see logs for git-sync side-car container:

```console
$ kubectl logs -l app=git-policy-importer -c git-sync -n nomos-system
```

## Monitoring

GKE Policy Management uses [Prometheus](https://prometheus.io/) to monitor the
various processes that comprise a GKE Policy Management deployment. These
processes include the NamespaceController, ResourceQuotaAdmissionController, and
others. Each process exports certain metrics that are available for you to
scrape from the configured port using Prometheus or any other tools you wish.

### GKE Policy Management Metrics

These are the [metrics](https://prometheus.io/docs/concepts/metric_types/) we
currently export:

Name                                                 | Type      | Labels                         | Description
---------------------------------------------------- | --------- | ------------------------------ | -----------
nomos_admission_controller_usage                     | Gauge     | app, policyspace, resource     | Policyspace quota usage per resource type
nomos_admission_controller_duration_seconds          | Histogram | app, namespace, allowed        | Admission duration distributions for apps such as resource quota
nomos_admission_controller_error_total               | Counter   | app, namespace                 | Total internal errors that occurred when reviewing admission requests
nomos_admission_controller_violations_total          | Counter   | app, policyspace, resource     | Policyspace quota violations per resource type
nomos_monitor_policies                               | Gauge     | state                          | Total number of policies (cluster and node) grouped by their sync status; should be similar to nomos_policy_importer_policy_nodes metric
nomos_monitor_last_import_timestamp                  | Gauge     |                                | Timestamp of the most recent import
nomos_monitor_last_sync_timestamp                    | Gauge     |                                | Timestamp of the most recent sync
nomos_monitor_sync_latency_seconds                   | Histogram |                                | Distribution of the latencies between importing and syncing each node
nomos_policy_importer_policy_node_operations_total   | Counter   | operation                      | Total operations that have been performed to keep policy node hierarchy up-to-date with source of truth
nomos_policy_importer_policy_nodes                   | Gauge     |                                | Number of policy nodes in current state
nomos_policy_importer_policy_state_transitions_total | Counter   | status                         | Total number of policy state transitions (A state transition can include changes to multiple resources)
nomos_syncer_error_total                             | Counter   | namespace, resource, operation | Total errors that occurred when executing syncer actions
nomos_syncer_event_timestamps                        | Gauge     | type                           | Timestamps when syncer events occurred
nomos_syncer_queue_size                              | Counter   |                                | Current size of syncer action queue
nomos_syncer_action_duration_seconds                 | Histogram | namespace, resource, operation | Syncer action duration distributions

### Scraping the Metrics

All metrics are available for scraping at port 8675. Prometheus includes a
process that you can optionally choose to
[run on your cluster](https://prometheus.io/docs/prometheus/latest/getting_started/)
alongside the GKE Policy Management processes. This process must be configured
to scrape the metrics which you are interested in.

Alternatively you can use
[Prometheus Operator](https://coreos.com/operators/prometheus/docs/latest/)
which is an abstraction layer provided by CoreOS to simplify conifguration of
metrics scraping. The following ServiceMonitor manifest will scrape all GKE
Policy Management metrics every 10 seconds:

```yaml
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: nomos
  namespace: monitoring
  labels:
    prometheus: kube-prometheus
spec:
  selector:
    matchLabels:
      monitored: "true"
  namespaceSelector:
    matchNames:
    - nomos-system
  endpoints:
  - port: metrics
    interval: 10s
```

### Example Queries

Number of pending syncer events in queue. If this grows to a large size it could
indicate an error where the cluster is not being synced from the source of
truth.

```console
nomos_syncer_queue_size
```

Count of errors from syncer upserts. Any metric that is named "error_total"
should not increase over time. In this example the query is using the operation
label to specifically look at upserts.

```console
nomos_syncer_error_total{operation="upsert"}
```

Syncer action latency, 90th percentile over last 10 minutes. If there is a
sustained increase in latency it could indicate a performance issue or be used
to diagnose other symptoms (such as an increase in the syncer queue size).

```console
histogram_quantile(0.9, rate(nomos_syncer_action_duration_seconds_bucket[10m]))
```

[< Back](../../README.md)
