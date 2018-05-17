# Using Nomos

## Policy Hierarchy Operations

### Creation

When using Git as source of truth, we represent the hierarchy of policyspaces
and namespaces using the filesystem hierarchy.

Following the [foo-corp example](concepts.md#example), we can have such a
directory structure ([Available on this GitHub
repo](https://github.com/frankfarzan/foo-corp-example)):

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
1.  A namespace directory can contain any number of Role and Rolebinding
    resources, and a single ResourceQuota resource.
1.  A namespace directory name must match the namespace name in all resources in
    that directory.
1.  A policyspace directory can contain any number of Rolebinding resources and
    a single ResourceQuota resource but must not contain Roles. These resources
    must not specify a namespace.
1.  The root policyspace directory can also contain any number of ClusterRole,
    ClusterRolebinding, and PodSecurityPolicy resources.
1.  Both policyspace and namespace directory names must be valid Kubernetes
    namespace names (i.e. [DNS
    Label](https://github.com/kubernetes/community/blob/master/contributors/design-proposals/architecture/identifiers.md))
    and must be unique in the hierarchy. In addition a name cannot be `default`,
    `nomos-system`, or have `kube-` prefix. Namespaces that match `kube-`,
    `nomos-system` and `default` wil be refered to as `Reserved Namespaces` for
    purposes of this document.

There are no requirements on file names or how many resources are packed in a
file. Any other file not explicitly mentioned above is ignored by Nomos in this
release (e.g. OWNERS files).

When a valid tree is committed to Git and synced, Nomos controllers
automatically create namespaces and corresponding policy resources to enforce
hierarchical policy. In this example, Nomos automatically creates
`shipping-dev`, `shipping-staging`, and `shipping-prod` namespaces. We discuss
specific policy types and their enforcement in later sections.

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
is destructive since it deletes that namespace and creates a new namespace.

Renaming a policyspace directory has no externally visible effect.

### Moving

Moving a policyspace or namespace directory can lead to policy changes in
namespaces, but does not delete a namespace or workload resources.

## Policy Types

### Namespace-level Policies

##### Role/Rolebinding

Nomos enables RBAC policies to be applied hierarchically following these
properties:

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
job-creators
pod-creators
```

Inheritance is implemented by flattening resources in namespaces. In
`shipping-dev` namespace, `pod-creators` is inherited and `job-creators` is
created directly in the namespace.

Note that policies are themselves resources which means a user may be able to
edit policies outside of Nomos (e.g. using kubectl) or create rolebindings
subject to [privilege escalation
prevention](https://kubernetes.io/docs/admin/authorization/rbac/#privilege-escalation-prevention-and-bootstrapping)
in Kubernetes.

##### ResourceQuota

A quota set on a namespace behaves just like it does in native kubernetes,
restricting the specified resources. In Nomos you can also set resource quota on
policyspaces. This will set the quota limit on all the namespaces that are
children of the provided policyspace within a single cluster. The policyspace
limit ensures that the sum of all the resources of a specified type in all the
children of the policyspace do not exceed the specified quota. Quota is
evaluated in a hierarchical fashion starting from the namespace, up the
policyspace hierarchy - this means that a quota violation at any level will
result in a Forbidden exception.

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
kubernetes cluster with the only addition being that Nomos will distribute and
manage them on the workload clusters.

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

### Kubectl

Since Nomos uses a filesystem tree of Kubernetes resources, `kubectl` can be
used to validate resource schemas. The following command recursively validates
all the resources in `foo-corp` directory without applying changes:

```console
$ kubectl apply -f foo-corp --recursive --dry-run
```

### Nomosvet

`nomosvet` is tool that validates a root policyspace directory against the
[constraints](#constraints) listed above.

To install nomosvet:

```console
$ curl https://storage.googleapis.com/nomos-release/nomosvet.sh -o nomosvet.sh
$ chmod +x nomosvet.sh
```

The following commands assume that you placed `nomosvet.sh` in a directory
mentioned in your `$PATH` environment variable.

You can manually run nomosvet:

```console
$ nomosvet.sh foo-corp
```

You can also automatically run nomosvet as a git [pre-commit
hook](https://git-scm.com/book/en/v2/Customizing-Git-Git-Hooks). In the root of
the repo, run:

```console
$ echo "nomosvet.sh foo-corp" > .git/hooks/pre-commit; chmod +x .git/hooks/pre-commit
```

You can also integrate this into your CI/CD setup, e.g. when using GitHub
[required status
check](https://help.github.com/articles/about-required-status-checks/).

## Guarantees

This section details the guarantees that Nomos makes based on the contents of
the git repo.

### On Install

**IMPORTANT: Do not install Nomos on a cluster with existing namespaces or
workloads**

During the install process, Nomos deletes all namespaces that have been created
on the cluster. We are presently working on a non destructive installation
process, and this document will be updated accordingly when the mechanisms are
avialable.

### Namespaces

Nomos will manage the lifecycle of all namespaces on a kubernetes cluster.

There's two categories of namespaces: Managed, and Reserved:

1.  **Reserved Namespaces** are the default namespaces that are installed on the
    kubernetes cluster (`kube-system`, `kube-public`, `default`) as well as the
    `nomos-system` namespace. These namespaces are exceptions and will not be
    managed by Nomos. Additionally, in order to be future proof, we also treat
    all `kube-` prefixed namespaces as `Reserved Namespaces`.
1.  **Managed Namesapces** are all the other namespaces on the cluster that are
    not Reserved Namespaces. These are created by creating a namespace directory
    and a namespace .yaml file in the git repo, and deleted by lack of their
    declaration in git.

Examples:

*   [foo-corp/audit/namespace.yaml](https://github.com/frankfarzan/foo-corp-example/blob/master/foo-corp/audit/namespace.yaml)
    will result in creation of the `audit` namespace with a parent `foo-corp`.
    Removal of this namespace.yaml will result in Nomos deleting the `audit`
    namespace.
*   [foo-corp/online/shipping-app-backend/shipping-prod/namespace.yaml](https://github.com/frankfarzan/foo-corp-example/blob/master/foo-corp/online/shipping-app-backend/shipping-prod/namespace.yaml)
    will result in creation of the shipping-prod namespace with ancestry
    [`shipping-app-backend`, `online`, `foo-corp`]. Deleting this namespace.yaml
    will result in deleting the `shipping-prod` namespace.

The following table describes the action that Nomos will take regarding a
namespace on the cluster.

Declared in git | Exists on cluster | Nomos Action
--------------- | ----------------- | ------------------------------------
true            | true              | no action
true            | false             | create namespace and manage policies
false           | true              | delete namespace from cluster
false           | false             | no action

### Cluster Scoped Policies

Policies in the cluster scope will be applied to the cluster exactly as they are
specified in the git repo. Existing resources at the cluster level will not be
managed unless a resource with the same name exists git.

Declared in git | On Cluster         | Nomos Action
--------------- | ------------------ | -----------------------------------
true            | matches git repo   | no action
true            | different than git | Nomos updates resource to match git
true            | does not exist     | Nomos creates resource from git
false           | exists             | no action
false           | does not exist     | no action

Examples:

*   ClusterRole `pod-accountant` exists on the cluster, but does not exist in
    git for [foo-corp](https://github.com/frankfarzan/foo-corp-example). Nomos
    is installed for foo-corp. Nomos will not delete or alter `pod-accountant`.
*   ClusterRole `namespace-reader` exists on the cluster, and exists in git for
    [foo-corp](https://github.com/frankfarzan/foo-corp-example). Nomos is
    installed for foo-corp. Nomos will now update `namespace-reader` to match
    the one declared in
    [namespace-reader-clusterrole.yaml](https://github.com/frankfarzan/foo-corp-example/blob/master/foo-corp/namespace-reader-clusterrole.yaml).
*   Nomos is installed for foo-corp. Someone adds a new ClusterRole
    `quota-viewer` to git in `foo-corp/quota-viewer-clusterrole.yaml`. Nomos
    will now create the `quota-viewer` ClusterRole matching the one in git. Time
    passes. Someone deletes the `quota-viewer-clusterrole.yaml` from git. Nomos
    will now remove `quota-viewer` from the cluster.

### Namespace Scoped Policies

Namespace scoped policies will be applied to match the intent of the source of
truth exactly as they are specified. This means that Nomos will overwrite or
delete any existing policies that do not match the declarations in the source of
truth.

Declared in git | On Cluster         | Nomos Action
--------------- | ------------------ | -----------------------------------
true            | matches git        | no action
true            | different than git | Nomos updates resource to match git
true            | does not exist     | Nomos creates resource from git
false           | resource exists    | Nomos deletes the resource
false           | does not exist     | no action

Examples:

*   RoleBinding
    [pod-creators](https://github.com/frankfarzan/foo-corp-example/blob/master/foo-corp/online/shipping-app-backend/pod-creator-rolebinding.yaml)
    is in git for foo-corp. Nomos will ensure that all `pod-creator`
    rolebindings in descendants of the `shipping-app-backend` policyspace
    (`shipping-prod`, `shipping-staging`, `shipping-dev`) exactly match the
    declared `pod-creator` RoleBinding. Time passes and someone modifies the
    [shipping-prod](https://github.com/frankfarzan/foo-corp-example/tree/master/foo-corp/online/shipping-app-backend/shipping-prod)
    `pod-creator` RoleBinding. Nomos will notice the change and update
    `pod-creator` to match the declaration in git. Time passes and someone
    removes `pod-creator` from git. Nomos will now remove the `pod-creator`
    resource from the descendant namespaces.
*   Someone creates a `secret-admin` Role in `shipping-prod`. Nomos will notice
    that the Role is not declared in `shipping-prod` or any of its ancestors and
    delete the `secret-admin` Role from the namespace.
*   Someone adds a `secret-admin` Role to git in `shipping-prod`. Nomos will
    notice the updated declarations and create the `secret-admin` role in the
    `shipping-prod` namespace.
