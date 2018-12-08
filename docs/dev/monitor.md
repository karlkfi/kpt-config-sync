# Monitoring

Please refer to the [monitoring section](../user/monitoring_and_debugging.md) of
the user guide for more details about the metrics that are available in GKE
Policy Management.

### Manually Inspecting Metrics

If you would like to inspect the current metric values manually, you can use
port forwarding to easily view them. For example let's say you want to view
metrics for the Syncer:

```console
$ kubectl port-forward -n nomos-system $(kubectl get pods -n nomos-system -l app=syncer -o jsonpath='{.items[0].metadata.name}') 8675
```

Now you can view the metric values by visiting http://localhost:8675/metrics. If you
have the Prometheus server process running then you can forward the port from it
(default is 9090) to enable local querying:

```console
$ kubectl port-forward -n monitoring $(kubectl get pods -n monitoring -l app=prometheus -o jsonpath='{.items[0].metadata.name}') 9090
```

Now you can use the
[HTTP API](https://prometheus.io/docs/prometheus/latest/querying/api/) to send
queries. For example try
http://localhost:9090/api/v1/query?query=nomos_policy_importer_policy_nodes.
