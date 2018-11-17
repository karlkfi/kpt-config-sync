# GKE Policy Management System Configuration

## Overview

The `system/` directory contains resources used to configure the GKE Policy
Management system.

## Repo

Exactly one Repo resource must be declared in the system directory.

For purposes of the example we have placed this object in nomos.yaml, however,
matching the file name is not a requirement. At the moment, the version must
match 1.0.0. If the semantics or format of the repo changes over time, this file
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
  version: "1.0.0"
```

## Sync

GKE Policy Management allows for syncing arbitrary kubernetes types from Git to
a Kubernetes cluster. Sync is configured by placing a Sync resource in the
**system** directory. The following example configures syncing RBAC types with
"inherit" mode (addressed later) for RoleBindings.

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
      mode: inherit
      versions:
      - version: v1
        compareFields:
        - subjects
        - roleRef
```

ResourceQuota has [special handling](rq.md). However, the Sync configuration is
not affected by this. The following shows an example of configuring sync on
ResourceQuota.

```console
$ cat system/resourcequota-sync.yaml
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

#### Standard mode

By default, policies have no inheritance and can only be set on a Namespace
directory. Placing a non-inherited policy in an Abstract Namespace directory
will cause an error.

To demonstrate, we move a Role from a Namespace directory to an Abstract
Namespace directory.

```console
$ mv rnd/new-prj/acme-admin-role.yaml rnd/
```

Now when we try to sync policies from our repo, we get the following error.

```console
Found issues: 1 error(s)

[1] KNV1007: Object "acme-admin" illegally declared in an Abstract Namespace directory. Move this object to a Namespace directory:

source: namespaces/rnd/acme-admin-role.yaml
metadata.name: acme-admin
group: rbac.authorization.k8s.io
apiVersion: v1
kind: Role
```

#### Inherited Mode

GKE Policy Management enables "inherited" policies to be applied hierarchically
following these properties:

1.  An "inherit" mode policy specified in a Abstract Namespace directory is
    inherited by all descendant namespaces
1.  An "inherit" mode policy can be specified in a Namespace directory (Existing
    K8S behavior)

For example, we can set the Sync mode for RoleBinding to inherit and create a
RoleBinding in the `shipping-app-backend` Abstract Namespace such that anyone
belonging to `shipping-app-backend-team` group is able to create pods in all
namespace descendants (i.e. `shipping-dev`, `shipping-staging`,
`shipping-prod`):

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

Note that GKE Policy Management is intended to be non-destructive to resources
that are created outside of the system which means a user may be able to edit
resources outside of GKE Policy Management (e.g. using kubectl) or create
rolebindings subject to
[privilege escalation prevention](https://kubernetes.io/docs/admin/authorization/rbac/#privilege-escalation-prevention-and-bootstrapping)
in Kubernetes.

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

## Reserved Namespaces

See [Managing Existing Clusters](existing_clusters.md).

Reserved namespaces are namespaces that GKE Policy Management will not manage.
This is here to indicates to GKE Policy Management should not allow another
namespace of the same name to be created via a Namespace directory as well as
suppresses alerts of an unknown namespace.

A ConfigMap defined in the root named "nomos-reserved-namespaces" defines the
reserved namespaces. For example, the following declares the namespaces
'test-sandbox', 'billing', and 'database' as reserved and as such they will be
ignored and not trigger warnings for namespaces that are not in the
declarations. Note that 'default', 'kube-system', 'kube-public' and
'nomos-system' do not need to be added to this list.

```console
$ cat system/nomos-reserved-namespaces.yaml
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

[< Back](../../README.md)
