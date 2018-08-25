# Git User Guide

GKE Policy Management supports using Git to centrally manage Namespaces and
policies across Kubernetes clusters. This *Policy as Code* approach ensures
policy configurations are:

*   __Immutable:__ A Git commit is an exact declaration of the desired state of
    policies.
*   __Auditable:__ Changes are reviewed and approved by administrators.
*   __Revertable:__ Misconfiguration is one of the most common reasons for
    service outages. It should be fast and easy to revert to a known good state.

In addition, the rich CI/CD ecosystem built around Git enables building
sophisticated pipelines for vetting and deploying at scale.

## Policy Hierarchy Operations

### Creation

When using Git as source of truth, we represent the hierarchy of policyspaces
and namespaces using the filesystem hierarchy.

Following the [foo-corp example](concepts.md#example), we can have such a
directory structure
([Available on this GitHub repo](https://github.com/frankfarzan/foo-corp-example)):

```console
foo-corp
|-- audit
|   `-- namespace.yaml
|-- online
|   `-- shipping-app-backend
|       |-- shipping-dev
|       |   |-- job-creator-rolebinding.yaml
|       |   |-- job-creator-role.yaml
|       |   |-- namespace.yaml
|       |   `-- quota.yaml
|       |-- shipping-prod
|       |   `-- namespace.yaml
|       |-- shipping-staging
|       |   `-- namespace.yaml
|       |-- pod-creator-rolebinding.yaml
|       `-- quota.yaml
|-- namespace-reader-clusterrolebinding.yaml
|-- namespace-reader-clusterrole.yaml
|-- pod-creator-clusterrole.yaml
|-- pod-security-policy.yaml
`-- viewers-rolebinding.yaml
```

##### Definitions

1.  A namespace directory is one that contains a Namespace resource.
1.  Any other directory is a policyspace.

##### Constraints

1.  A namespace directory must be a leaf directory (i.e., it must not have any
    children).
1.  A namespace directory can contain any number of uniquely named Role and
    Rolebinding resources, and a single ResourceQuota resource.
1.  A namespace directory name must match the namespace name in all resources in
    that directory.
1.  A policyspace directory can contain any number of uniquely named Rolebinding
    resources and a single ResourceQuota resource but must not contain Roles.
    These resources must not specify a namespace.
1.  The root policyspace directory can also contain any number of uniquely named
    ClusterRole, ClusterRolebinding, and PodSecurityPolicy resources.
1.  Both policyspace and namespace directory names must be valid Kubernetes
    namespace names (i.e.
    [DNS Label](https://github.com/kubernetes/community/blob/master/contributors/design-proposals/architecture/identifiers.md))
    and must be unique in the hierarchy. In addition a name cannot be `default`,
    `nomos-system`, or have `kube-` prefix. Namespaces that match `kube-*`,
    `nomos-system` and `default` are a special class of namespaces called
    `Reserved Namespaces` that GKE Policy Management will not interact with.
    This topic is discussed in depth in the
    [namespaces user guide](git_user_guide_namespaces.md)

There are no requirements on file names or how many resources are packed in a
file. Any other file not explicitly mentioned above is ignored by GKE Policy
Management in this release (e.g. OWNERS files).

When a valid tree is committed to Git and synced, GKE Policy Management
controllers automatically create namespaces and corresponding policy resources
to enforce hierarchical policy. In this example, GKE Policy Management
automatically creates `shipping-dev`, `shipping-staging`, and `shipping-prod`
namespaces. We discuss specific policy types and their enforcement in later
sections.

Note that when using Git as source of truth, it is up to the repo owners to set
proper access control mechanism (e.g. using OWNERS or CODEOWNER files) to ensure
right people can approve/review/commit policy changes. It is recommended to use
a hierarchical access control mechanism such as OWNERS file in order to delegate
policy changes instead of requiring a central authority to approve all changes.

### Deletion

Deleting a namespace directory is a very destructive operation. All resources
including identities, policies and workload resources will be deleted on every
cluster where this namespace is present. Similarly deleting a policyspace
directory recursively, deletes all descendаnt names and associated resources.

### Renaming

Renaming a namespace directory (which requires renaming Namespace name as well)
is destructive since it **deletes that namespace and creates a new namespace**.

Renaming a policyspace directory has no externally visible effect.

### Moving

Moving a policyspace or namespace directory can lead to policy changes in
namespaces, but does not delete a namespace or workload resources.

### Existing Namespaces

GKE Policy Management will not manage namespaces that already exist on a cluster
at install time. For details on how to configure namespaces that already exist,
please see the [namespaces user guide](git_user_guide_namespaces.md)

## Policy Types

### Namespace-level Policies

##### Role/Rolebinding

GKE Policy Management enables RBAC policies to be applied hierarchically
following these properties:

1.  A RoleBinding specified in a policyspace is inherited by all descendant
    namespaces
1.  A Role cannot be specified in a policyspace. If multiple namespaces need to
    refer to the same role, use a ClusterRole.
1.  A RoleBinding can be specified in a namespace (Existing K8S behavior)
1.  A Role can be specified in a namespace (Existing K8S behavior).

For example, we can create a RoleBinding in `shipping-app-backend` policyspace
such that anyone belonging to `shipping-app-backend-team` group is able to
create pods in all namespace descendants (i.e. `shipping-dev`,
`shipping-staging`, `shipping-prod`):

```console
$ cat foo-corp/online/shipping-app-backend/pod-creator-rolebinding.yaml
```

```yaml
kind: RoleBinding
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: pod-creators
subjects:
- kind: Group
  name: shipping-app-backend-team
  apiGroup: rbac.authorization.k8s.io
roleRef:
  kind: ClusterRole
  name: pod-creator
  apiGroup: rbac.authorization.k8s.io
```

This is done by automatically creating inherited RoleBindings in a namespace:

```console
$ kubectl get rolebinding --namespace shipping-dev -o name
shipping-dev.job-creators
shipping-app-backend.pod-creators
foo-corp.viewers
```

Inheritance is implemented by flattening resources in namespaces. In
`shipping-dev` namespace, `pod-creators` is inherited and `job-creators` is
created directly in the namespace. While inheriting, the rolebindinding
resources have the directory name prepended with a dot separator. This is to
allow a rolebinding to be created at any level without naming conflicts.

Note that policies are themselves resources which means a user may be able to
edit policies outside of GKE Policy Management (e.g. using kubectl) or create
rolebindings subject to
[privilege escalation prevention](https://kubernetes.io/docs/admin/authorization/rbac/#privilege-escalation-prevention-and-bootstrapping)
in Kubernetes.

##### ResourceQuota

A quota set on a namespace behaves just like it does in native kubernetes,
restricting the specified resources. In GKE Policy Management you can also set
resource quota on policyspaces. This will set the quota limit on all the
namespaces that are children of the provided policyspace within a single
cluster. The policyspace limit ensures that the sum of all the resources of a
specified type in all the children of the policyspace do not exceed the
specified quota. Quota is evaluated in a hierarchical fashion starting from the
namespace, up the policyspace hierarchy - this means that a quota violation at
any level will result in a Forbidden exception.

A quota is allowed to be set to immediately be in violation. For example, when a
workload namespace has 11 pods, we can still set quota to `pods: 10` in a parent
policyspace, creating an overage. If a workload namespace is in violation, the
ResourceQuotaAdmissionController will prevent new objects of that type from
being created until the total object count falls below the quota limit, but
existing objects will still be valid and operational.

Here we add hard quota limit on number of pods across all namespaces having
`shipping-app-backend` as an ancestor:

```console
$ cat foo-corp/online/shipping-app-backend/quota.yaml
```

```yaml
kind: ResourceQuota
apiVersion: v1
metadata:
  name: pod-quota
spec:
  hard:
    pods: "3"
```

In this case, total number of pods allowed in `shipping-prod`, `shipping-dev`,
and `shipping-staging` is 3. When creating the fourth pod (e.g. in
`shipping-prod`), you will see the following error:

```console
Error from server (Forbidden): exceeded quota in policyspace "shipping-app-backend", requested: pods=4, limit: pods=3
```

### Cluster-level Policies

Cluster-level policies will function in the same manner as in a vanilla
kubernetes cluster with the only addition being that GKE Policy Management will
distribute and manage them on the workload clusters.

Cluster-level policies must be placed immediately within the root policyspace
directory. Since cluster-level policies have far-reaching effect, they should
only be editable by cluster admins.

##### ClusterRole/ClusterRoleBinding

For example, we can create namespace-reader ClusterRole:

```console
$ cat foo-corp/namespace-viewer-role.yaml
```

```yaml
kind: ClusterRole
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: namespace-reader
rules:
- apiGroups: [""]
  resources: ["namespaces"]
  verbs: ["get", "watch", "list"]
```

And a ClusterRoleBinding referencing this Role:

```console
$ cat foo-corp/namespace-viewer-rolebinding.yaml
```

```yaml
kind: ClusterRoleBinding
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: namespace-readers
subjects:
- kind: User
  name: cheryl@foo-corp.com
  apiGroup: rbac.authorization.k8s.io
roleRef:
  kind: ClusterRole
  name: namespace-reader
  apiGroup: rbac.authorization.k8s.io
```

##### PodSecurityPolicy

PodSecurityPolicies are created in the same manner as other cluster level
resources:

```console
$ cat foo-corp/pod-security-policy.yaml
```

```yaml
apiVersion: extensions/v1beta1
kind: PodSecurityPolicy
metadata:
  name: psp
spec:
  privileged: false
  seLinux:
    rule: RunAsAny
  supplementalGroups:
    rule: RunAsAny
  runAsUser:
    rule: RunAsAny
  fsGroup:
    rule: RunAsAny
  volumes:
  - '*'
```

## Validation

Before committing policy configuration in Git and pushing changes to Kubernetes
clusters, it is important to validate them first.

`nomosvet` is tool that validates a root policyspace directory against the
[constraints](#constraints) listed above as well as validating resources using
their schema (Similar to `kubectl apply --dry-run`).

To install nomosvet:

```console
$ curl https://storage.googleapis.com/nomos-release/stable/linux_amd64/nomosvet -o nomosvet
$ chmod u+x nomosvet
```

You can replace `linux_amd64` in the URL with other supported platforms:

*   `darwin_amd64`
*   `windows_amd64`

The following commands assume that you placed `nomosvet` in a directory
mentioned in your `$PATH` environment variable.

You can manually run nomosvet:

```console
$ nomosvet foo-corp
```

You can also automatically run nomosvet as a git
[pre-commit hook](https://git-scm.com/book/en/v2/Customizing-Git-Git-Hooks). In
the root of the repo, run:

```console
$ echo "nomosvet foo-corp" > .git/hooks/pre-commit; chmod +x .git/hooks/pre-commit
```

You can also integrate this into your CI/CD setup, e.g. when using GitHub
[required status check](https://help.github.com/articles/about-required-status-checks/).

## Guarantees

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
    policyspace (`shipping-prod`, `shipping-staging`, `shipping-dev`) exactly
    match the declared `pod-creator` RoleBinding. Time passes and someone
    modifies the
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
