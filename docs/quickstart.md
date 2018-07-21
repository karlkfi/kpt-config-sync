# GKE Policy Management Quickstart

This quickstart shows a step-by-step guide to install GKE Policy Management on
Kubernetes clusters and create and synchronize hierarchical policies from a Git
repository.

## Installation

Follow the [Installation Guide](installation.md) to install GKE Policy
Management on one or more Kuberentes clusters.

## Creating hierarchical policies

Once GKE Policy Management components are deployed and running in a cluster,
namespaces will be automatically created. For the
[foo-corp example](https://github.com/frankfarzan/foo-corp-example):

```console
$ kubectl get namespaces -l nomos-managed
```

This should return 4 namespaces: `shipping-dev`,
`shipping-staging`,`shipping-prod`, and `audit`.

Rolebindings are inherited from parent directories:

```console
$ kubectl get rolebinding -n shipping-dev
```

This should return 3 rolebindings: `job-creators`,
`shipping-app-backend.pod-creators`, and `foo-corp.viewers`.

You can test effective RBAC policies by impersonating users. For example, this
should be forbidden:

```console
$ kubectl get secrets -n shipping-dev --as bob@foo-corp.com
```

whereas, this should succeed since `bob@foo-corp.com` has the `pod-creator`
role:

```console
$ kubectl get pods -n shipping-dev --as bob@foo-corp.com
```

To see inherited ResourceQuota in action, we can create a pod and request
resources that exceed the limits set in the parent directory:

```console
$ cat <<EOF | kubectl create --as bob@foo-corp.com -f -
```

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: busybox-sleep
  namespace: shipping-prod
spec:
  containers:
  - name: busybox
    image: busybox
    args:
    - sleep
    - "1000000"
    resources:
      requests:
        memory: "64Mi"
        cpu: "2"
EOF
```

Try changing the cpu request from `2` to `200m`. This time it should succeed.

## What's next

[GKE Policy Management User Guide](git_user_guide.md)
