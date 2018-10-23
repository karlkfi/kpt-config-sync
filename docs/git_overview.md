# Git Overview

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
├── cluster
│   ├── namespace-reader-clusterrolebinding.yaml
│   ├── namespace-reader-clusterrole.yaml
│   ├── pod-creator-clusterrole.yaml
│   └── pod-security-policy.yaml
├── namespaces
│   ├── foo-corp
│   │   ├── audit
│   │   │   └── namespace.yaml
│   │   ├── online
│   │   │   └── shipping-app-backend
│   │   │       ├── pod-creator-rolebinding.yaml
│   │   │       ├── quota.yaml
│   │   │       ├── shipping-dev
│   │   │       │   ├── job-creator-rolebinding.yaml
│   │   │       │   ├── job-creator-role.yaml
│   │   │       │   ├── namespace.yaml
│   │   │       │   └── quota.yaml
│   │   │       ├── shipping-prod
│   │   │       │   └── namespace.yaml
│   │   │       └── shipping-staging
│   │   │           └── namespace.yaml
│   │   └── viewers-rolebinding.yaml
│   ├── sre-rolebinding.yaml
│   └── sre-supported-selector.yaml
└── system
    ├── nomos.yaml
    ├── podsecuritypolicy.yaml
    ├── rbac.yaml
    └── resourcequota.yaml
```

#### Directories and files

1.  **cluster** contains cluster-scoped resources
1.  **namespaces** is the root of the namespace hierarchy. Namespace and
    namespace-scoped resources are declared within this directory and its
    subdirectories.
1.  **system** contains configuration related to syncing, the repo and reserved
    namespaces.

#### Definitions

The following definitions are regarding directories within the 'namespaces'
directory.

1.  A namespace directory is one that contains a Namespace resource.
1.  Any other directory is a policyspace.

#### Constraints

##### cluster directory

1.  The cluster directory can contain any number of uniquely named cluster
    scoped resources. Any namespace-scoped resources in this directory will be
    treated as an error.
1.  Users may not place namespace-scoped resources in this directory. These are
    placed in the namespaces hierarchy.
1.  Users may not place nomos.dev/v1alpha1 {Sync,Nomos} objects in this
    directory

##### namespaces directory

The namespace hierarchy exists within the namespaces directory. This is composed
of directories that represent namespaces that will be created on the cluster and
intermediate policyspace directories which represent common policy attach points
for descendant namespaces.

1.  A Namespace object must only exist in leaf directories (i.e., it must not
    have any children)
    1.  A directory with a Namespace object is a namespace directory
    1.  If a Namespace object exists, its name must match the directory name.
    1.  A namespace directory can contain any number of uniquely named
        resources, but only a single ResourceQuota resource.
    1.  All resources declared in a namespace directory must specify a namespace
        name that matches the name of the Namespace resource in the directory.
1.  A policyspace directory can contain any number of hierarchical resources but
    only a single ResourceQuota resource. These resources must not specify a
    namespace.
1.  Both policyspace and namespace directory names must be valid Kubernetes
    namespace names (i.e.
    [DNS Label](https://github.com/kubernetes/community/blob/master/contributors/design-proposals/architecture/identifiers.md))
    and must be unique in the hierarchy. In addition a name cannot be `default`,
    `nomos-system`, or have `kube-` prefix. Namespaces that match `kube-*`,
    `nomos-system` and `default` are a special class of namespaces called
    `Reserved Namespaces` that GKE Policy Management will not interact with.
    This topic is discussed in depth in the
    [namespaces user guide](git_namespaces.md)

##### system directory

1.  The system directory must only contain the nomos.dev objects Nomos and Sync,
    and an optional ConfigMap (core v1) that contains the reserved namespace
    mapping.

##### File Naming

There are no requirements on file names or how many resources are packed in a
file. Any other file not explicitly mentioned above is ignored by GKE Policy
Management in this release (e.g. OWNERS files).

When a valid namespace hierarchy is committed to Git and synced, GKE Policy
Management controllers automatically create namespaces and corresponding policy
resources to enforce hierarchical policy. In this example, GKE Policy Management
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
please see the [namespaces user guide](git_namespaces.md)

## Configuring Nomos

Exactly one Nomos resource must be declared in the system directory. For
purposes of the example we have placed this object in nomos.yaml, however,
matching the file name is not a requirement. At the moment, the repoVersion must
match 1.0.0. If the semantics or format of the repo changes over time, this file
will be the mechanism used to determine compatibility and automate upgrade.

```console
$ cat system/nomos.yaml
```

```yaml
kind: NomosConfig
apiVersion: nomos.dev/v1alpha1
metadata:
  name: config
spec:
  repoVersion: "1.0.0"
```

## Configuring Kubernetes Resource Sync

Nomos allows for syncing arbitrary kubernetes types from Git to a Kubernetes
cluster. Sync is configured by placing a Sync resource in the **system**
directory. The following example configures syncing RBAC types with "inherit"
mode (addressed later) for RoleBindings.

```console
$ cat system/rbac.yaml
```

```yaml
kind: Sync
apiVersion: nomos.dev/v1alpha1
metadata:
  name: rbac
spec:
  groups:
  - group: rbac.authorization.k8s.io
    kinds:
    - kind: ClusterRole
      versions:
      - version: v1
    - kind: ClusterRoleBinding
      versions:
      - version: v1
    - kind: Role
      versions:
      - version: v1
    - kind: RoleBinding
      mode: inherit
      versions:
      - version: v1
```

ResourceQuota has special handling (as described below), however, the Sync
configuration is not affected by this. The following shows an example of
configuring sync on ResourceQuota.

```console
$ cat system/resourcequota.yaml
```

```yaml
kind: Sync
apiVersion: nomos.dev/v1alpha1
metadata:
  name: resourcequotas
spec:
  groups:
  - kinds:
    - kind: ResourceQuota
      versions:
      - version: v1
```

## Policy Types

### Namespace-level Policies

#### Standard mode

#### Inherited Mode

Nomos enables "inherited" policies to be applied hierarchically following these
properties:

1.  An "inherit" mode policy specified in a policyspace is inherited by all
    descendant namespaces
1.  An "inherit" mode policy can be specified in a namespace (Existing K8S
    behavior)

For example, we can set the Sync mode for RoleBinding to inherit and create a
RoleBinding in the `shipping-app-backend` policyspace such that anyone belonging
to `shipping-app-backend-team` group is able to create pods in all namespace
descendants (i.e. `shipping-dev`, `shipping-staging`, `shipping-prod`):

```console
$ cat namespaces/online/shipping-app-backend/pod-creator-rolebinding.yaml
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
viewers
```

Inheritance is implemented by flattening resources in namespaces. In
`shipping-dev` namespace, `pod-creators` is inherited and `job-creators` is
created directly in the namespace.

Note that Nomos is intended to be non-destructive to resources that are created
outside of the system which means a user may be able to edit resources outside
of Nomos (e.g. using kubectl) or create rolebindings subject to
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
$ cat namespaces/online/shipping-app-backend/quota.yaml
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

##### ClusterRole/ClusterRoleBinding Example

For example, we can create namespace-reader ClusterRole:

```console
$ cat cluster/namespace-viewer-role.yaml
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
$ cat cluster/namespace-viewer-rolebinding.yaml
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

##### PodSecurityPolicy Example

PodSecurityPolicies are created in the same manner as other cluster level
resources:

```console
$ cat cluster/pod-security-policy.yaml
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
