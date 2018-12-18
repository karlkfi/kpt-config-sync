# ClusterSelectors

In the [Overview](overview.md) section, we described how the resources specified
in the `namespaces/` and `cluster/` directories get distributed to all enrolled
clusters. This provides a way to address any Kubernetes resource to an entire
fleet of clusters. Sometimes it is necessary to address Kubernetes resources to
specific clusters only. This can be achieved using `ClusterSelectors`.

Let's take a look at a `foo-corp` example that uses this option. The example is
similar to the one used for [NamespaceSelectors](namespaceselectors.md). A new
directory is introduced, called `clusterregistry/`.

```console
foo-corp
├── cluster
│   ├── namespace-reader-clusterrolebinding.yaml
│   ├── namespace-reader-clusterrole.yaml
│   ├── pod-creator-clusterrole.yaml
│   └── pod-security-policy.yaml
├── clusterregistry
│   ├── cluster-1.yaml
│   ├── cluster-2.yaml
│   ├── clusterselector-1.yaml
│   └── clusterselector-2.yaml
├── namespaces
│   ├── audit
│   │   └── namespace.yaml
│   ├── online
│   │   └── shipping-app-backend
│   │       ├── pod-creator-rolebinding.yaml
│   │       ├── quota.yaml
│   │       ├── shipping-dev
│   │       │   ├── job-creator-rolebinding.yaml
│   │       │   ├── job-creator-role.yaml
│   │       │   ├── namespace.yaml
│   │       │   └── quota.yaml
│   │       ├── shipping-prod
│   │       │   └── namespace.yaml
│   │       └── shipping-staging
│   │           └── namespace.yaml
│   ├── sre-rolebinding.yaml
│   ├── sre-supported-selector.yaml
│   └── viewers-rolebinding.yaml
└── system
    ├── nomos.yaml
    ├── podsecuritypolicy-sync.yaml
    ├── rbac-sync.yaml
    └── resourcequota-sync.yaml
```

To define Kubernetes resources addressed to specific clusters, follow these
steps:

##### 1. At install time, give Nomos-wide names to your clusters

To use `ClusterSelectors`, clusters must be named at
[installation](installation.md) time by adding the parameter `spec.clusterName`
into Nomos resource.

```console
$ cat nomos1.yaml
```

```yaml
apiVersion: addons.sigs.k8s.io/v1alpha1
kind: Nomos
metadata:
  name: nomos
  namespace: nomos-system
spec:
  clusterName: cluster-1
  git:
    syncRepo: git@github.com:frankfarzan/foo-corp-example.git
    syncBranch: 0.1.0
    secretType: ssh
    policyDir: foo-corp
  enableHierarchicalResourceQuota: true
```

`spec.clusterName` is an installation-wide cluster name.

Repeat the cluster naming and installation steps for all named clusters.

```console
$ cat nomos2.yaml
```

```yaml
apiVersion: addons.sigs.k8s.io/v1alpha1
kind: Nomos
metadata:
  name: nomos
  namespace: nomos-system
spec:
  clusterName: cluster-2
  git:
    syncRepo: git@github.com:frankfarzan/foo-corp-example.git
    syncBranch: 0.1.0
    secretType: ssh
    policyDir: foo-corp
  enableHierarchicalResourceQuota: true
```

##### 2. Add labels to your clusters

Clusters are labeled so they can be grouped.

```console
$ cat clusterregistry/cluster-1.yaml
```

```yaml
kind: Cluster
apiVersion: clusterregistry.k8s.io/v1alpha1
metadata:
  name: cluster-1
  labels:
    environment: prod
```

```console
$ cat clusterregistry/cluster-2.yaml
```

```yaml
kind: Cluster
apiVersion: clusterregistry.k8s.io/v1alpha1
metadata:
  name: cluster-2
  labels:
    environment: dev
```

##### 3. Group your clusters using `ClusterSelector`

Clusters are grouped so they can be treated as a unit. Each `ClusterSelector`
may match zero or more clusters.

```console
$ cat clusterregistry/clusterselector-1.yaml
```

```yaml
kind: ClusterSelector
apiVersion: nomos.dev/v1alpha1
metadata:
  name: selector-env-prod
spec:
  selector:
    matchLabels:
      environment: prod
```

```console
$ cat clusterregistry/clusterselector-2.yaml
```

```yaml
kind: ClusterSelector
apiVersion: nomos.dev/v1alpha1
metadata:
  name: selector-env-dev
spec:
  selector:
    matchLabels:
      environment: dev
```

##### 4. Annotate the object that you want to target with the selector

This makes the object named `quota` available only on clusters covered by the
selector named `selector-env-prod`.

```console
$ cat namespaces/online/shipping-app-backend/quota.yaml
```

```yaml
kind: ResourceQuota
apiVersion: v1
metadata:
  name: quota
  annotations:
    nomos.dev/cluster-selector: selector-env-prod
spec:
  hard:
    pods: "3"
    cpu: "1"
    memory: 1Gi
```

Adding the `ClusterSelector` annotation to a Namespace resource makes the
namespace and all included objects available only on clusters covered by the
matching selector.

```console
$ cat namespaces/online/shipping-app-backend/shipping-dev.yaml
```

```yaml
apiVersion: v1
kind: Namespace
metadata:
  name: shipping-dev
  annotations:
    nomos.dev/cluster-selector: selector-env-dev
```

`ClusterSelector` annotation can be applied to Kubernetes resources in the
`cluster/` directory.

```console
$ cat cluster/pod-security-policy.yaml
```

```yaml
apiVersion: extensions/v1beta1
kind: PodSecurityPolicy
metadata:
  name: psp
  annotations:
    nomos.dev/cluster-selector: selector-env-prod
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

[< Back](../../README.md)
