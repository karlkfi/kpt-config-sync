# Managing Existing Clusters

## Summary

GKE Policy Management tracks whether it should manage a namespace by using
labels. During a normal creation workflow, GKE Policy Management applies this
label to the namespace automatically, however, namespaces that already exist on
the cluster at install time will not have the proper labeling to ensure that
resources are not accidentally deleted during the install.

## Namespace Labeling

The following label values indicate the action that GKE Policy Management will
take for managing a namespace.

Label                                   | GKE Policy Management Action
--------------------------------------- | ----------------------------
none                                    | No management
nomos.dev/namespace-management=policies | Manage policies for the namespace
nomos.dev/namespace-management=full     | Manage policies and lifecycle of the namespace

## Types of Namespaces

There's three categories of namespaces: Managed, Reserved, and Legacy
Namepsaces:

1.  **Reserved Namespaces** are the default namespaces that are installed on the
    kubernetes cluster (`kube-system`, `kube-public`, `default`) as well as the
    `nomos-system` namespace. These namespaces are exceptions and will not be
    managed by GKE Policy Management. Additionally, in order to be future proof,
    we also treat all `kube-` prefixed namespaces as `Reserved Namespaces`.
1.  **Managed Namesapces** are namespaces on the cluster that are fully managed
    by GKE Policy Management. They all have the management label
    nomos.dev/namespace-management=full and exist in the Git source of truth as
    well as on the cluster. They are created when added to Git, and deleted when
    removed from Git.
1.  **Legacy Namesapces** are namespaces on the cluster without the
    nomos.dev/namespace-management=full label. They will cause alerts for being
    in a non-ideal state, however, GKE Policy Management will neither manage nor
    delete them. They can be converted to a managed namespace by following a
    migration process (Pending Docs / Tooling).

Examples:

*   [foo-corp/audit/namespace.yaml](https://github.com/frankfarzan/foo-corp-example/blob/master/foo-corp/audit/namespace.yaml)
    will result in creation of the `audit` namespace with a parent `foo-corp`.
    Removal of this namespace.yaml will result in GKE Policy Management deleting
    the `audit` namespace.
*   [foo-corp/online/shipping-app-backend/shipping-prod/namespace.yaml](https://github.com/frankfarzan/foo-corp-example/blob/master/foo-corp/online/shipping-app-backend/shipping-prod/namespace.yaml)
    will result in creation of the shipping-prod namespace with ancestry
    [`shipping-app-backend`, `online`, `foo-corp`]. Deleting this namespace.yaml
    will result in deleting the `shipping-prod` namespace.

### Configuring Reserved Namespaces

Reserved namespaces are namespaces that GKE Policy Management will not manage.
This is here to indicate to GKE Policy Management should not allow another
namespace of the same name to be created via a namespace directory as well as
suppressing alerts of an unknown namespace.

A ConfigMap defined in the root named "nomos-reserved-namespaces" defines the
reserved namespaces. For example, the following declares the namespaces
'test-sandbox', 'billing', and 'database' as reserved and as such they will be
ignored and not trigger warnings for namespaces that are not in the
declarations. Note that 'default', 'kube-system', 'kube-public' and
'nomos-system' do not need to be added to this list.

```console
$ cat foo-corp/nomos-reserved-namespaces.yaml
```

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: nomos-reserved-namespaces
data:
  test-sandbox: reserved
  billing: reserved
  database: reserved
```

### Namespace evaluation

The following table describes the action that GKE Policy Management will take
regarding a namespace on the cluster.

Declared in Git | Exists on Cluster | Namespace Label                         | GKE Policy Management Action   | Alert Triggered
--------------- | ----------------- | --------------------------------------- | ------------------------------ | ---------------
true            | false             | N/A                                     | create namespace               | None
true            | true              | N/A                                     | none                           | Namespace declared but not managed
true            | true              | nomos.dev/namespace-management=policies | manage policies                | Partially managed namespace
true            | true              | nomos.dev/namespace-management=full     | manage policies                | None
false           | true              | No label                                | none                           | Unknown namespace
false           | true              | nomos.dev/namespace-management=policies | none                           | Unknown namespace
false           | true              | nomos.dev/namespace-management=full     | deletes namespace from cluster | None
