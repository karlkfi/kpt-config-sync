# Monitoring and Debugging

## Logging

Nomos follows [K8S logging
convention](https://github.com/kubernetes/community/blob/master/contributors/devel/logging.md).
By default, all binaries log at V(2).

List all nomos-system pods:

```
kubectl get deployment -n nomos-system
NAME                                 DESIRED   CURRENT   UP-TO-DATE   AVAILABLE   AGE
git-policy-importer                  1         1         1            1           13d
resourcequota-admission-controller   1         1         1            1           9d
syncer                               1         1         1            1           13d
```

To see logs for pod:

```
kubectl logs -l app=syncer --namespace nomos-system
```

git-policy-importer pod has two containers.

To see logs for policy-importer container:

```shell
kubectl logs -l app=git-policy-importer -c policy-importer -n nomos-system
```

To see logs for git-sync side-car container:

```shell
kubectl logs -l app=git-policy-importer -c git-sync -n nomos-system
```

## Monitoring

TBD
