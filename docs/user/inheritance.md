# Configuring Inheritance

By default, most namespace-scoped resources must be placed in Namespace
directories and are not allowed in Abstract Namespace directories. The two
exceptions are `RoleBindings` and `ResourceQuota`, as described in
[GKE Policy Management System Configuration](system_config.md). It's possible to
change this inheritance behavior.

## Example

Inheritance can be added to any type by setting the `hierarchyMode` field
in the corresponding Sync:

```console
$ cat system/rbac.yaml

kind: Sync
apiVersion: nomos.dev/v1alpha1
metadata:
  name: rbac
spec:
  groups:
  - group: rbac.authorization.k8s.io
    kinds:
    - kind: Role
      hierarchyMode: inherit
      versions:
      - version: v1
        compareFields:
        - rules
```

To demonstrate the effect, we return to the foo-corp example. In
[System Configuration](system_config.md), we tried to move
`job-creator-role.yaml` to an Abstract Namespace, and that resulted in an error.
We'll do it again, this time with inheritance enabled:

```console
$ mv namespaces/online/shipping-app-backend/shipping-dev/job-creator-role.yaml namespaces/online/
```

Once the above changes are committed, check the cluster:

```console
$ kubectl get role --all-namespaces | grep job-creator
shipping-dev       job-creator                                  1m
shipping-prod      job-creator                                  1m
shipping-staging   job-creator                                  1m
```

The role has been instantiated in all descendant namespaces.

## Modes

`hierarchyMode` can take the following values: `inherit`, `hierarchicalQuota`,
or `none`. It can also be omitted.

### Inherit mode

This is the mode used in the example. If specified, resources are allowed in
Abstract Namespaces. There, they are flattened into descendant namespaces. This
matches the default behavior for `RoleBindings`, as described in
[System Configuration](system_config.md). `RoleBindings` still default to this
behavior if `hierarchyMode` is unspecified.

Just as in default `RoleBinding` inheritance, conflicts are avoided by
disallowing duplicates in the same ancestry.

### hierarchicalQuota mode

`hierarchyMode: hierarchicalQuota` is only allowed for `ResourceQuota`, and it
matches the default behavior described in [Hierarchical ResourceQuota](rq.md).
If `hierarchyMode` is omitted, `ResourceQuota` still follows that same behavior.
So, `hieararchicalQuota` isn't strictly necessary, but you may choose to set it
to make the behavior explicit.

`ResourceQuota` may also use `inherit` or `none` modes.

### None mode

Specifying `hierarchyMode: none` disables inheritance. It may be used with any
type, including `RoleBinding` and `ResourceQuota`. When inheritance is disabled,
resources are not allowed in Abstract Namespaces.

`none` is the same as the default behavior for all other resources.

### Compatibility

The following chart summarizes defaults and allowed values for `hierarchyMode`
for different resource types:

Resource      | Default value     | Allowed values
------------- | ----------------- | --------------------------------
ResourceQuota | hierarchicalQuota | hierarchicalQuota, inherit, none
RoleBinding   | inherit           | inherit, none
others        | none              | inherit, none
