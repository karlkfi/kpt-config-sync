# Management Flow

The following decision tree shows the expected operations taken by GKE Policy
Management System based on the desired state in the Git repo and the current
state of the cluster, including the
[management labels](git_existing_clusters.md) applied by the user.

![drawing](img/system_flow.png)

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
