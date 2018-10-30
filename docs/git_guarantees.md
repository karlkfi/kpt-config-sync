# System Guarantees

This section details the guarantees that GKE Policy Management makes based on
the contents of the git repo and what exists on cluster.

## Cluster Scoped Policies

Policies in the cluster scope will be applied to the cluster exactly as they are
specified in the git repo. Existing resources at the cluster level will not be
managed unless a resource with the same name exists git.

Declared in git | On Cluster         | GKE Policy Management Action
--------------- | ------------------ | ----------------------------
true            | matches git repo   | no action
true            | different than git | GKE Policy Management updates resource to match git
true            | does not exist     | GKE Policy Management creates resource from git
false           | exists             | no action
false           | does not exist     | no action

Examples:

*   ClusterRole `pod-accountant` exists on the cluster, but does not exist in
    git for [foo-corp](https://github.com/frankfarzan/foo-corp-example). GKE
    Policy Management is installed for foo-corp. GKE Policy Management will not
    delete or alter `pod-accountant`.
*   ClusterRole `namespace-reader` exists on the cluster, and exists in git for
    [foo-corp](https://github.com/frankfarzan/foo-corp-example). GKE Policy
    Management is installed for foo-corp. GKE Policy Management will now update
    `namespace-reader` to match the one declared in
    [namespace-reader-clusterrole.yaml](https://github.com/frankfarzan/foo-corp-example/blob/master/foo-corp/namespace-reader-clusterrole.yaml).
*   GKE Policy Management is installed for foo-corp. Someone adds a new
    ClusterRole `quota-viewer` to git in
    `foo-corp/quota-viewer-clusterrole.yaml`. GKE Policy Management will now
    create the `quota-viewer` ClusterRole matching the one in git. Time passes.
    Someone deletes the `quota-viewer-clusterrole.yaml` from git. GKE Policy
    Management will now remove `quota-viewer` from the cluster.

## Namespace Scoped Policies

Namespace scoped policies will be applied to match the intent of the source of
truth exactly as they are specified. This means that GKE Policy Management will
overwrite or delete any existing policies that do not match the declarations in
the source of truth.

Declared in git | On Cluster         | GKE Policy Management Action
--------------- | ------------------ | ----------------------------
true            | matches git        | no action
true            | different than git | GKE Policy Management updates resource to match git
true            | does not exist     | GKE Policy Management creates resource from git
false           | resource exists    | GKE Policy Management deletes the resource
false           | does not exist     | no action

Examples:

*   RoleBinding
    [pod-creators](https://github.com/frankfarzan/foo-corp-example/blob/master/foo-corp/online/shipping-app-backend/pod-creator-rolebinding.yaml)
    is in git for foo-corp. GKE Policy Management will ensure that all
    `pod-creator` rolebindings in descendants of the `shipping-app-backend`
    Abstract Namespace (`shipping-prod`, `shipping-staging`, `shipping-dev`)
    exactly match the declared `pod-creator` RoleBinding. Time passes and
    someone modifies the
    [shipping-prod](https://github.com/frankfarzan/foo-corp-example/tree/master/foo-corp/online/shipping-app-backend/shipping-prod)
    `pod-creator` RoleBinding. GKE Policy Management will notice the change and
    update `pod-creator` to match the declaration in git. Time passes and
    someone removes `pod-creator` from git. GKE Policy Management will now
    remove the `pod-creator` resource from the descendant namespaces.
*   Someone creates a `secret-admin` Role in `shipping-prod`. GKE Policy
    Management will notice that the Role is not declared in `shipping-prod` or
    any of its ancestors and delete the `secret-admin` Role from the namespace.
*   Someone adds a `secret-admin` Role to git in `shipping-prod`. GKE Policy
    Management will notice the updated declarations and create the
    `secret-admin` role in the `shipping-prod` namespace.
