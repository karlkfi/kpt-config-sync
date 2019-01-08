# GKE Policy Management System Configuration

## Overview

The `system/` directory contains resources used to configure the GKE Policy
Management system.

## Repo

Exactly one Repo resource must be declared in the system directory.

For purposes of the example we have placed this object in nomos.yaml, however,
matching the file name is not a requirement. At the moment, the version must
match 0.1.0. If the semantics or format of the repo changes over time, this file
will be the mechanism used to determine compatibility and automate upgrade.

```console
$ cat system/nomos.yaml
```

```yaml
kind: Repo
apiVersion: nomos.dev/v1alpha1
metadata:
  name: repo
spec:
  version: "0.1.0"
```

## Sync

GKE Policy Management allows for syncing arbitrary kubernetes types from Git to
a Kubernetes cluster. Sync is configured by placing a Sync resource in the
`system/` directory. The following example configures syncing RBAC types.

When syncing resources from Git and comparing them with the current cluster, we
need some criteria to determine if a resource in Git matches what is on the
cluster. The way we do this is by checking if the labels and annotations match
and if the contents of a set of field(s) in the resources match. The set of
field(s) to compare against can be set by the user and are defined in Sync
resources. By default, comparison is done against the `spec` field, which most
resource have
[by convention](https://github.com/eBay/Kubernetes/blob/master/docs/devel/api-conventions.md#spec-and-status).
Since the RBAC resources specified below do not have `spec` fields, we are
specifying the relevant comparison fields below.

```console
$ cat system/rbac-sync.yaml
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
        compareFields:
        - rules
    - kind: ClusterRoleBinding
      versions:
      - version: v1
        compareFields:
        - subjects
        - roleRef
    - kind: Role
      versions:
      - version: v1
        compareFields:
        - rules
    - kind: RoleBinding
      versions:
      - version: v1
        compareFields:
        - subjects
        - roleRef
```

### Inheritance

GKE Policy Management allows `RoleBindings` and `ResourceQuota` to be placed in
Abstract Namespace directories, and have those policies instantiated in
descendant Namespaces.

#### RoleBinding inheritance

GKE Policy Management provides inheritance for `RoleBindings` specially,
following these properties:

1.  A `RoleBinding` specified in an Abstract Namespace directory is inherited by
    all descendant namespaces.
1.  A `RoleBinding` can be specified in a Namespace directory, just like any
    other resource.

For example, we can create a RoleBinding in the `shipping-app-backend` Abstract
Namespace such that anyone belonging to `shipping-app-backend-team` group is
able to create pods in all namespace descendants (i.e. `shipping-dev`,
`shipping-staging`, `shipping-prod`):

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

GKE Policy Management automatically creates inherited RoleBindings in the
descendant Namespaces:

```console
$ kubectl get rolebinding --namespace shipping-dev -o name
job-creators
pod-creators
viewers
```

Inheritance is implemented by flattening resources in namespaces. In
`shipping-dev` namespace, `pod-creators` is inherited and `job-creators` is
created directly in the namespace.

What happens if a hierarchy contains conflicting `RoleBindings` (i.e. multiple
`RoleBindings` with the same name)? For simplicity, GKE Policy Management
disallows that. I.e., it is an error for a `RoleBinding` to have the same name
as another `RoleBinding` either in the same directory or in any ancestor
Abstract Namespace.

Note that GKE Policy Management is intended to be non-destructive to resources
that are created outside of the system which means a user may be able to edit
resources outside of GKE Policy Management (e.g. using kubectl) or create
RoleBindings subject to
[privilege escalation prevention](https://kubernetes.io/docs/admin/authorization/rbac/#privilege-escalation-prevention-and-bootstrapping)
in Kubernetes.

#### ResourceQuota inheritance

Like `RoleBindings`, `ResourceQuotas` may also appear in Abstract Namespaces.
`ResourceQuota` inheritance has some unique behaviors, described fully in
[Hierarchical ResourceQuota](rq.md).

#### Custom Resources

GKE Policy Management does not handle syncing
[CustomResourceDefinitions (CRDS)](https://kubernetes.io/docs/tasks/access-kubernetes-api/custom-resources/custom-resource-definitions/).
However, it does handle syncing the corresponding Custom Resources. This means
that the CRDs themselves need to be added to the cluster out of band. Deleting a
CRD will also delete all the Custom Resources on the cluster and GKE Policy
Management never deletes resources that it does not explicitly manage.

So we first need to add the CRD to the cluster.

```console
$ cat <<EOF | kubectl apply -f -
apiVersion: apiextensions.k8s.io/v1beta1
kind: CustomResourceDefinition
metadata:
  name: bar.foo-corp.com
spec:
  group: foo-corp.com
  version: v1
  scope: Namespaced
  names:
    plural: bars
    singular: bar
    kind: Bar
  validation:
    openAPIV3Schema:
      properties:
        spec:
          type: object
          required:
          - bar
          properties:
            bar:
              type: string
EOF
```

Then we add a corresponding Sync in the system directory.

```console
$ cat <<EOF > system/bar-sync.yaml
kind: Sync
apiVersion: nomos.dev/v1alpha1
metadata:
  name: bars
spec:
  groups:
  - group: foo-corp.com
    kinds:
    - kind: Bar
      versions:
      - version: v1
EOF
```

Since this is a namespaced-scoped resource, we add resources to a directory
under the Namespace directory. If it were cluster scoped, we would just add the
resources to the clusters directory.

```console
$ cat <<EOF > namespaces/online/shipping-app-backend/shipping-dev/bar.yaml
apiVersion: foo-corp.com/v1
kind: Bar
metadata:
  name: example-bar
spec:
  bar: baz
EOF
```

### Other Resource Types

Only `RoleBindings` and `ResourceQuota` are allowed in Abstract Namespaces.
Putting any other resource in an Abstract Namespace causes an error.

To demonstrate, in foo-corp, we move a Role from a Namespace directory to an
Abstract Namespace directory:

```console
$ mv namespaces/online/shipping-app-backend/shipping-dev/job-creator-role.yaml namespaces/online/
```

Now when we try to sync policies from our repo, we get the following error.

```console
Found issues: 1 error(s)

[1] KNV1007: Object "job-creator" illegally declared in an Abstract Namespace directory. Move this object to a Namespace directory:

source: namespaces/online/job-creator-role.yaml
metadata.name: job-creator
group: rbac.authorization.k8s.io
apiVersion: v1
kind: Role
```

#### Modifying Syncs and Resources Simultaneously

It is discouraged to have a git commit that alters both Syncs and the
corresponding resources. Since there is a dependency between Syncs and
resources, the intent behind toggling resource management and altering resources
is ambiguous in certain cases.

An example illustrating this can be found
[here](management_flow.md#sync-and-resource-precedence).

[< Back](../../README.md)
