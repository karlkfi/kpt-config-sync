# GKE Policy Management System Configuration

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

???

#### Inherited Mode

GKE Policy Management enables "inherited" policies to be applied hierarchically
following these properties:

1.  An "inherit" mode policy specified in a Abstract Namespace directory is
    inherited by all descendant namespaces
1.  An "inherit" mode policy can be specified in a namespace (Existing K8S
    behavior)

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

## Reserved Namespaces

See [Managing Existing Clusters](git_namespaces.md).

Reserved namespaces are namespaces that GKE Policy Management will not manage.
This is here to indicates to GKE Policy Management should not allow another
namespace of the same name to be created via a namespace directory as well as
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
