# Using Nomos

## Policy Hierarchy Operations

### Creation

When using Git as source of truth, we represent the hierarchy of policyspaces
and namespaces using the filesystem hierarchy.

Following the [foo-corp example](concepts.md#example) above, we can have such a
directory structure ([Available on this GitHub
repo](https://github.com/frankfarzan/foo-corp-example)):

```
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
1.  A policyspace directory must not contain a Namespace resource.
1.  A policyspace directory can contain any number of Rolebinding resources and
    a single ResourceQuota resource but must not contain Roles. These resources
    must not specify a namespace.
1.  The root policyspace directory can also contain any number of ClusterRole,
    ClusterRolebinding, and PodSecurityPolicy resources.
1.  Both policyspace and namespace directory names must be valid Kubernetes
    namespace names (i.e. [DNS
    Label](https://github.com/kubernetes/community/blob/master/contributors/design-proposals/architecture/identifiers.md))
    and must be unique in the hierarchy. In addition a name cannot be "default",
    "nomos-system", or have "kube-" prefix.
1.  Any other file not explicitly mentioned above is ignored by Nomos in this
    release (e.g. OWNERS files).

There are no requirements on file names or how many resources are packed in a
file.

When a valid tree is committed to Git and synced, Nomos controllers
automatically create namespaces and corresponding policy resources to enforce
hierarchical policy. In this example, Nomos automatically creates
"shipping-dev", "shipping-staging", and "shipping-prod" namespaces. We discuss
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

For example, we can create a RoleBinding in "shipping-app-backend" policyspace
such that anyone belonging to "shipping-app-backend-team" group is able to
create pods in all namespace descendants (i.e. "shipping-dev",
"shipping-staging", "shipping-prod"):

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
"shipping-dev" namespace, "pod-creators" is inherited and "job-creators" is
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
workload namespace has 11 pods, we can still set quota to "pods: 10" in a parent
policyspace, creating an overage. If a workload namespace is in violation, the
ResourceQuotaAdmissionController will prevent new objects of that type from
being created until the total object count falls below the quota limit, but
existing objects will still be valid and operational.

Here we add hard quota limit on number of pods across all namespaces having
"shipping-app-backend" as an ancestor:

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

In this case, total number of pods allowed in "shipping-prod", "shipping-dev",
and "shipping-staging" is 3. When creating the fourth pod (e.g. in
"shipping-prod"), you will see the following error:

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

You can manually run nomosvet:

```console
$ nomosvet.sh foo-corp
```

This requires nomosvet.sh is in your PATH.

You can also automatically run nomosvet as a git [pre-commit
hook](https://git-scm.com/book/en/v2/Customizing-Git-Git-Hooks). In the root of
the repo, run:

```console
$ echo "nomosvet.sh foo-corp" > .git/hooks/pre-commit; chmod +x .git/hooks/pre-commit
```

You can also integrate this into your CI/CD setup, e.g. when using
GitHub [required status
check](https://help.github.com/articles/about-required-status-checks/).
