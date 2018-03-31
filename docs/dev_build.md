# Building

To build and deploy all Nomos binaries:

```shell
cd $NOMOS
make NOMOS_INSTALLER_CONFIG=path/to/config deploy
```

This should be sufficent to successfully deploy all components.

Rest of the section discusses individual components.

## Syncer

Create policynodes:

```shell
kubectl apply -f examples/acme/policynodes/acme.yaml
```

List the created policynodes:

```shell
kubectl get pn
```

List the corresponding namespaces:

```shell
kubectl get ns
```

See the syncer logs:

```shell
kubectl logs -l app=syncer --namespace nomos-system
```

## ResourceQuota Admission Controller Webhook

List resourcequota objects:

```shell
kubectl get --all-namespaces quota
```

See the controller logs:

```shell
kubectl logs -l app=resource-quota --namespace nomos-system
```

To test the hierarchical quota check, try creating configmaps.

This one should work:

```shell
kubectl create configmap map1 --namespace new-prj
```

This one will be blocked thereafter due to not enough quota.

```shell
kubectl create configmap map2 --namespace new-prj
```

## GitPolicyImporter

To see logs for policy-importer container:

```shell
kubectl logs -l app=git-policy-importer -c policy-importer -n nomos-system
```

To see logs for git-sync side-car container:

```shell
kubectl logs -l app=git-policy-importer -c git-sync -n nomos-system
```

## Nomos kubectl plugin

This installs Nomos-specific commands as a `kubectl` plugin.

```shell
make install-kubectl-plugin
```

After installation, this command gets the namespace hierarchy starting from the
namespace `frontend`.

```shell
kubectl plugin nomos get namespaces --namespace=frontend
```

The `--as` impersonation works as well:

```shell
kubectl plugin nomos get namespaces --namespace=frontend --as=alice@acme.com
```

This command gets all applicable Nomos quota objects.

```shell
kubectl plugin nomos get quota --namespace=frontend
```

This command gets all roles across namespaces.

```shell
kubectl plugin nomos get roles --namespace=frontend
```

This command gets all role bindings across namespaces.

```shell
kubectl plugin nomos get rolebindings --namespace=frontend
```
