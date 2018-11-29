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
centralized policy management. Currently, there is support for Git.

## First Steps

Try it out by following the [Quickstart Guide](docs/user/quickstart.md).

## Installing GKE Policy Management

*   [System Overview](docs/user/system_overview.md)
*   [Installation](docs/user/installation.md)

## Using GKE Policy Management

*   [Overview](docs/user/overview.md)
*   [NamespaceSelectors](docs/user/namespaceselectors.md)
*   [Management Flow](docs/user/management_flow.md)
*   [System Configuration](docs/user/system_config.md)
*   [Hierarchical ResourceQuota](docs/user/rq.md)
*   [Monitoring and Debugging](docs/user/monitoring_and_debugging.md)

## Contributing to GKE Policy Management

[Developer Guide](docs/dev/guide.md)

## Changelog

[CHANGELOG](docs/CHANGELOG.md)

## Send feedback

Questions or feedback? Get in touch with us at
[nomos-support@google.com](mailto:nomos-support@google.com).
