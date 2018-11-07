# System Guarantees

This section details the guarantees that GKE Policy Management makes based on
the contents of the git repo and what exists on cluster.

## Management Actions Taken

Based on what is specified in the git repo and what exists on the Kubernetes
cluster, there are a variety of criteria that determine whether or not resources
are created or existing resources are updated/deleted.

These include:

1.  **Sync Declared**: Whether a Sync was declared in the git repo for a given
    resource.
1.  **Resource Declared**: Whether a resource was declared in the git repo.
1.  **On Cluster**: Whether a resource currently exists on the Kubernetes
    cluster.
1.  **Has Label**: Whether the resource on the Kubernetes cluster has a
    `nomos.dev/managed` label applied.
1.  **Resource Matches**: Whether the resource declared in git and the
    Kubernetes resource both match according to the fields specified in the
    declared Sync.

A comprehensive table of actions taken depending on the git repo and the state
of the cluster is listed below. The main takeaways are that GKE Policy
Management will only delete or update existing Kubernetes resources that have a
Sync declared in the git repo, have a corresponding resource in the git repo and
the corresponding existing Kubernetes resource has a management label applied.
It will only create new resources when a Sync is declared, a resource exists in
the git repo and there is no existing matching resource on the Kubernetes
cluster.

Sync Declared | Resource Declared | On Cluster | Has Label | Resources Matches | GKE Policy Management Action
------------- | ----------------- | ---------- | --------- | ----------------- | ----------------------------
yes           | yes               | yes        | yes       | yes               | no action
yes           | yes               | yes        | yes       | no                | GKE Policy Management updates resource to match git
yes           | yes               | yes        | no        | N/A               | no action
yes           | yes               | no         | N/A       | N/A               | GKE Policy Management creates resource to match git
yes           | no                | yes        | yes       | N/A               | GKE Policy Management deletes resource to match git
yes           | no                | yes        | no        | N/A               | no action
yes           | no                | no         | N/A       | N/A               | no action
no            | N/A               | N/A        | N/A       | N/A               | no action

Examples:

*   ClusterRole `pod-accountant` exists on the cluster, but does not exist in
    git for [foo-corp](https://github.com/frankfarzan/foo-corp-example). GKE
    Policy Management is installed for foo-corp and has a
    [Sync](https://github.com/frankfarzan/foo-corp-example/blob/master/foo-corp/system/rbac-sync.yaml)
    for ClusterRole. GKE Policy Management will not delete or alter
    `pod-accountant`.
*   GKE Policy Management is installed for foo-corp. Someone adds a new
    ClusterRole `quota-viewer` to git in
    `foo-corp/cluster/quota-viewer-clusterrole.yaml`. GKE Policy Management will
    now create the `quota-viewer` ClusterRole matching the one in git. Time
    passes. Someone deletes the `quota-viewer-clusterrole.yaml` from git. GKE
    Policy Management will now remove `quota-viewer` from the cluster.
*   Role `job-creator` exists on the cluster in shipping-dev namespace with a
    `nomos.dev/managed` label applied and exists in git for
    [foo-corp](https://github.com/frankfarzan/foo-corp-example). GKE Policy
    Management is installed for foo-corp and has a
    [Sync](https://github.com/frankfarzan/foo-corp-example/blob/master/foo-corp/system/rbac-sync.yaml)
    for Role. GKE Policy Management will now update `job-creator` to match the
    one declared in
    [job-creator-role.yaml](https://github.com/frankfarzan/foo-corp-example/blob/master/foo-corp/namespaces/online/shipping-app-backend/shipping-dev/job-creator-role.yaml).
*   RoleBinding
    [pod-creators](https://github.com/frankfarzan/foo-corp-example/blob/master/foo-corp/namespaces/online/shipping-app-backend/pod-creator-rolebinding.yaml)
    is in git for foo-corp and a
    [Sync](https://github.com/frankfarzan/foo-corp-example/blob/master/foo-corp/system/rbac-sync.yaml)
    has been declared Rolebinding. GKE Policy Management will ensure that all
    `pod-creator` rolebindings in descendants of the `shipping-app-backend`
    Abstract Namespace (`shipping-prod`, `shipping-staging`, `shipping-dev`)
    exactly match the declared `pod-creator` RoleBinding. Time passes and
    someone modifies the
    [shipping-prod](https://github.com/frankfarzan/foo-corp-example/tree/master/foo-corp/namespaces/online/shipping-app-backend/shipping-prod)
    `pod-creator` RoleBinding. GKE Policy Management will notice the change and
    update `pod-creator` to match the declaration in git. Time passes and
    someone removes `pod-creator` from git. GKE Policy Management will now
    remove the `pod-creator` resource from the descendant namespaces.
*   Foo-corp has a
    [Sync](https://github.com/frankfarzan/foo-corp-example/blob/master/foo-corp/system/rbac-sync.yaml)
    declared for Role. Someone creates a `secret-admin` Role in `shipping-prod`.
    GKE Policy Management will notice that the Role is not declared in
    `shipping-prod` or any of its ancestors, but will not delete it because it
    does not have a `nomos.dev/managed` label applied on it. Later on, the
    `nomos.dev/managed` label is added ot it. GKE Policy Management will now
    delete the `secret-admin` Role from the namespace.
*   Foo-corp has a
    [Sync](https://github.com/frankfarzan/foo-corp-example/blob/master/foo-corp/system/rbac-sync.yaml)
    declared for Role. Someone adds a `shipping-admin` Role to git in
    `shipping-prod`. GKE Policy Management will notice the updated declarations
    and create the `shipping-admin` role in the `shipping-prod` namespace.
