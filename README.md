# Google Kubernetes Engine Policy Management

GKE Policy Management (aka Nomos) enables policy distribution and hierarchical
policy enforcement for multi-tenant multi-cluster Kubernetes deployments.

In environments with many users spread across many teams, having multiple
tenants within a cluster, allocated into namespaces, maximizes resource
utilization while providing isolation. However, as the number of namespaces
grows, it becomes increasingly hard for cluster operators to manage
per-namespace policies like authorization (RBAC Roles/Rolebinding) and quota
(ResourceQuota), etc. In addition, real world deployments often require multiple
clusters in order to tolerate region failures, reduce network latencies for end
users, or simply to scale beyond the current size limits of a Kubernetes
cluster. GKE Policy Management makes it easier to manage large multi-tenant and
multi-cluster deployments, by reducing the load on cluster operators and
reducing the surface area to secure.

GKE Policy Management can be extended to support different sources of truth for
centralized policy management. Currently, there is support for Git and Google
Cloud Platform.

## First Steps

Watch the [demo](https://storage.googleapis.com/nomos-release/demo.mp4).

Try it out by following the [Quickstart Guide](docs/quickstart.md).

## Using GKE Policy Management

*   [System Overview](docs/system_overview.md)
*   [Installation](docs/installation.md)
*   Importing from Git:
    *   [Overview](docs/git_overview.md)
    *   [Validation](docs/git_validation.md)
    *   [ResourceQuota](docs/rq.md)
    *   [NamespaceSelectors](docs/git_namespaceselectors.md)
    *   [Managing Existing Clusters](docs/git_namespaces.md)
    *   [System Guarantees](docs/git_guarantees.md)
*   Importing from GCP:
    *   [Overview](docs/gcp_overview.md)
*   [Monitoring and Debugging](docs/monitoring_and_debugging.md)

## Contributing to GKE Policy Management

[Developer Guide](docs/dev/guide.md)

## Send feedback

Questions or feedback? Get in touch with us at
[nomos-support@google.com](mailto:nomos-support@google.com).
