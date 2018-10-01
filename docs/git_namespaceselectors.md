# NamespaceSelectors

In the [Overview](git_overview.md) section, we described how resources such as
RoleBindings are inherited hierarchically. This provides a powerful mechanism to
apply policies to all Namespaces in a subtree. However, some organizations
require the flexibility to apply policies to just a subset of the Namespaces
based on label selection. This _cross-cutting_ functionality can be used instead
of, or as a complement, to hierarchical policy evaluation.

Let's take a look at the foo-corp example again:

```console
foo-corp
├── audit
│   └── namespace.yaml
├── online
│   └── shipping-app-backend
│       ├── shipping-dev
│       │   ├── job-creator-rolebinding.yaml
│       │   ├── job-creator-role.yaml
│       │   ├── namespace.yaml
│       │   └── quota.yaml
│       ├── shipping-prod
│       │   └── namespace.yaml
│       ├── shipping-staging
│       │   └── namespace.yaml
│       ├── pod-creator-rolebinding.yaml
│       └── quota.yaml
├── namespace-reader-clusterrolebinding.yaml
├── namespace-reader-clusterrole.yaml
├── pod-creator-clusterrole.yaml
├── pod-security-policy.yaml
└── viewers-rolebinding.yaml
```

`audit` and `shipping-prod` namespaces contain workloads that are deployed in
production. As a result, we want to give the prod SRE team access to resources
in these namespaces. We can do this using `NamespaceSelector` objects. Follow
these steps:

##### 1. Add labels to Namespace objects

```console
$ cat foo-corp/audit/namespace.yaml
```

```yaml
apiVersion: v1
kind: Namespace
metadata:
  name: audit
  labels:
    env: prod
```

```console
$ cat foo-corp/online/shipping-app-backend/shipping-prod/namespace.yaml
```

```yaml
apiVersion: v1
kind: Namespace
metadata:
  name: shipping-prod
  labels:
    env: prod
```

##### 2. Add NamespaceSelector object

```console
$ cat foo-corp/sre-supported-selector.yaml
```

```yaml
kind: NamespaceSelector
apiVersion: nomos.dev/v1alpha1
metadata:
  name: sre-supported
spec:
  selector:
    matchLabels:
      env: prod
```

`NamespaceSelector` is a CustomResourceDefinition that includes a
[LabelSelector][1] field. The existing Label and LabelSelector mechanism in
Kubernetes is applied. An example of a more complex LabelSelector would be:

```yaml
kind: NamespaceSelector
apiVersion: nomos.dev/v1alpha1
metadata:
  name: prod-restrict
spec:
  selector:
    matchLabels:
      environment: prod
    matchExpressions:
      - {key: privacy, operator: In, values: [sensitive, restricted]}

```

##### 3. Add a RoleBinding with corresponding annotation

```console
$ cat foo-corp/sre-rolebinding.yaml
```

```yaml
kind: RoleBinding
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: sre-admin
  annotations:
    nomos.dev/namespace-selector: sre-supported
subjects:
- kind: Group
  name: sre@foo-corp.com
  apiGroup: rbac.authorization.k8s.io
roleRef:
  kind: ClusterRole
  name: admin
  apiGroup: rbac.authorization.k8s.io
```

Note the following:

*   Annotation `nomos.dev/namespace-selector` refers to the name of the
    `NamespaceSelector` object.
*   `NamespaceSelector` object must be placed in the same directory where it is
    referenced.

Given the above set up, `sre-admin` Rolebindings will only be created in `audit`
and `shipping-prod` namespaces.

[1]: https://kubernetes.io/docs/concepts/overview/working-with-objects/labels
