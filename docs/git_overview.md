# Overview

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

## Table of Contents

1.  [Filesystem Standard](#filesystem-standard)
    1.  [namespaces/](#namespaces)
    1.  [cluster/](#cluster)
    1.  [system/](#system)
1.  [Filesystem Operations](#filesystem-operations)
    1.  [Creation](#creation)
    1.  [Deletion](#deletion)
    1.  [Rename](#rename)
    1.  [Move](#move)

## Filesystem Standard

GKE Policy Management Filesystem Standard defines the directory structure and
file contents. This is analogous to the Linux Filesystem Hierarchy Standard, and
is a natural way to operate on hierarchical resources without requiring a
complicated domain specific language.

For example, we can have such a directory structure
([Available on this GitHub repo](https://github.com/frankfarzan/foo-corp-example)):

```console
foo-corp
├── cluster
│   ├── namespace-reader-clusterrolebinding.yaml
│   ├── namespace-reader-clusterrole.yaml
│   ├── pod-creator-clusterrole.yaml
│   └── pod-security-policy.yaml
├── namespaces
│   ├── audit
│   │   └── namespace.yaml
│   ├── online
│   │   └── shipping-app-backend
│   │       ├── shipping-dev
│   │       │   ├── job-creator-rolebinding.yaml
│   │       │   ├── job-creator-role.yaml
│   │       │   ├── namespace.yaml
│   │       │   └── quota.yaml
│   │       ├── shipping-prod
│   │       │   └── namespace.yaml
│   │       ├── shipping-staging
│   │       │   └── namespace.yaml
│   │       ├── pod-creator-rolebinding.yaml
│   │       └── quota.yaml
│   ├── sre-rolebinding.yaml
│   ├── sre-supported-selector.yaml
│   └── viewers-rolebinding.yaml
└── system
    ├── nomos.yaml
    ├── podsecuritypolicy-sync.yaml
    ├── rbac-sync.yaml
    └── resourcequota-sync.yaml
```

We define the semantics of each directory below:

### namespaces/

`namespaces` directory is the root of the namespace hierarchy and contains
namespace-scoped resources (e.g. RBAC RoleBindings).

Kubernetes does not natively provide a hierarchy of namespaces (namespaces are
flat). GKE Policy Management implements a hierarchy of namespaces to enable
management of a large number of resources across many teams.

There are two types of sub-directories in `namespaces`:

*   __Namespace directory:__ A directory containing a Namespace resource. A
    Namespace directory is a one-to-one mapping to a Kubernetes [Namespace][1].
*   __Abstract Namespace directory:__ Any other directory. Conceptually, an
    Abstract Namespace directory represents an intermediate node in the
    namespace hierarchy.

As an example of how `shipping-prod` Namespace is declared:

```console
$ cd foo-corp
$ cat namespaces/online/shipping-app-backend/shipping-prod/namespace.yaml
```

```yaml
apiVersion: v1
kind: Namespace
metadata:
  name: shipping-prod
  labels:
    env: prod
```

In foo-corp, our hierarchy looks like this:

![drawing](img/foo_corp_hierarchy.png)

By modeling the hierarchy like this, we enable the Shipping App Backend team to
manage three different namespaces while only have to maintain one authorization
policy for team members. Each of their namespaces is isolated by environment,
allowing identically-named objects in the three envionments' instantiations of
the backend stack, as well as providing tighter security, e.g. allowing one
namespace to have additional authorized users but not the others, and allocating
private quota to each namespace.

The following constraints apply to `namespaces` directory and are enforced
during [validation](git_validation.md):

1.  A Namespace directory MUST be a leaf directory.
1.  A Namespace directory's name MUST match `metadata.name` field of the
    contained Namespace resource.
1.  A Namespace directory MAY contain any number of uniquely named
    namespace-scoped resources.
1.  An Abstract Namespace directory MAY contain any number of hierarchical
    resources.
1.  Resources MUST NOT specify `metadata.namespace` field as it is inferred
    automatically.
1.  All directory names MUST be valid Kubernetes namespace names (i.e.
    [DNS Label](https://github.com/kubernetes/community/blob/master/contributors/design-proposals/architecture/identifiers.md)).
    In addition a name MUST NOT be `default`, `nomos-system`, or have `kube-`
    prefix. This topic is discussed in depth in the
    [namespaces user guide](git_namespaces.md).
1.  All directory names MUST be unique in the hierarchy.

### cluster/

`cluster` directory contains cluster-scoped resources (e.g. RBAC
ClusterRolebindings).

The following constraints apply to `cluster` directory and are enforced during
[validation](git_validation.md):

1.  The cluster directory MAY contain any number of uniquely named
    cluster-scoped resources.
1.  The cluster directory MUST NOT contain namespace-scoped resources.

### system/

`system` directory contains resources for configuring the GKE Policy Management
system.

The `system` directory MUST only contain the `nomos.dev` objects, and an
optional ConfigMap (core v1) that contains the reserved namespace mapping. See
[GKE Policy Management System Configuration](system_config.md).

## Filesystem Operations

### Creation

When a valid namespace hierarchy is committed to Git and synced, GKE Policy
Management controllers automatically creates namespaces and corresponding policy
resources to enforce hierarchical policy. In the foo-corp example, GKE Policy
Management automatically creates `audit`, `shipping-dev`, `shipping-staging`,
and `shipping-prod` namespaces. We discuss specific policy types and their
enforcement in later sections.

Note that when using Git as source of truth, it is up to the repo owners to set
proper access control mechanism (e.g. using OWNERS or CODEOWNER files) to ensure
right people can approve/review/commit policy changes. It is recommended to use
a hierarchical access control mechanism such as OWNERS file in order to delegate
policy changes instead of requiring a central authority to approve all changes.

### Deletion

Deleting a Namespace directory is a very destructive operation. All resources
including identities, policies and workload resources will be deleted on every
cluster where this namespace is present. Similarly deleting an Abstract
Namespace directory, deletes all descendant names and associated resources.

### Rename

Renaming a Namespace directory (which requires renaming Namespace name as well)
is destructive since it **deletes that namespace and creates a new namespace in
Kubernetes**.

Renaming an Abstract Namespace directory has no externally visible effect.

### Move

Moving a Namespace or an Abstract Namespace directory can lead to policy changes
in namespaces, but does not delete a namespace or workload resources.

[1]: https://kubernetes.io/docs/concepts/overview/working-with-objects/namespaces/
