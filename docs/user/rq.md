# Hierarchical ResourceQuota

A ResourceQuota object in a Namespace directory behaves just like it does in
native kubernetes, restricting the specified resources.

In Nomos you can also set resource quota on Abstract Namespaces. This will set
the quota limit on all the namespaces that are children of the provided Abstract
Namespace within a single cluster. The Abstract Namespace limit ensures that the
sum of all the resources of a specified type in all the children of the Abstract
Namespace do not exceed the specified quota. Quota is evaluated in a
hierarchical fashion starting from the namespace, up the Abstract Namespace
hierarchy - this means that a quota violation at any level will result in a
Forbidden exception.

To enable hierarchical ResourceQuota enforcement, set the
`enableHierarchicalResourceQuota` flag to true during
[installation](installation.md#create-the-nomos-config-file)

A quota can be set to immediately be in violation. For example, when a workload
namespace has 11 pods, we can still set quota to `pods: 10` in a parent Abstract
Namespace, creating an overage. If a workload namespace is in violation, the
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
Error from server (Forbidden): exceeded quota in Abstract Namespace "shipping-app-backend", requested: pods=4, limit: pods=3
```

## Caveats

Hierarchical ResourceQuota does not support the
[Quota Scopes](https://kubernetes.io/docs/concepts/policy/resource-quotas/#quota-scopes)
or
[Priority Class](https://kubernetes.io/docs/concepts/policy/resource-quotas/#limit-priority-class-consumption-by-default)
features. Usage of these features will result in undefined behavior.

[< Back](../../README.md)
