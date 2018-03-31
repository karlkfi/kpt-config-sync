# Nomos

Nomos enables policy distribution and hierarchical policy enforcement for
multi-tenant multi-cluster Kubernetes deployments.

In environments with many users spread across many teams, having multiple
tenants within a cluster, allocated into namespaces, maximizes resource
utilization while providing isolation. However, as the number of namespaces
grows, it becomes increasingly hard for cluster operators to manage
per-namespace policies like authorization (RBAC Roles/Rolebinding) and quota
(ResourceQuota), etc. In addition, real world deployments often require multiple
clusters in order to tolerate region failures, reduce network latencies for end
users, or simply to scale beyond the current size limits of a Kubernetes
cluster. Nomos makes it easier to manage large multi-tenant and multi-cluster
deployments, by reducing the load on cluster operators and reducing the surface
area to secure.

## Try Nomos

See [Quickstart](docs/quickstart.md).

## Using Nomos

*   [Concepts](docs/concepts.md)
*   [System Overview](docs/system_overview.md)
*   [Installation](docs/installation.md)
*   [User Guide](docs/user_guide.md)
*   [Monitoring and Debugging](docs/monitoring_and_debugging.md)

## Contributing to Nomos

[Developer Guide](docs/dev_guide.md)

## Send feedback

Questions or feedback? Get in touch with us at
[nomos-support@google.com](mailto:nomos-support@google.com).
