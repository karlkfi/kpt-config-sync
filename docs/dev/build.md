# Building

To build and deploy all GKE Policy Management binaries,
first create a yaml configuration file, following either
the [Git Configuration](../git_config.md) or
[GCP Configuration](../gcp_config.md) instructions, according
to your choice of source of truth. Then:

```console
$ make installer-staging
$ cd .output/staging/installer
$ ./install.sh --config=PATH_TO_YOUR_CONFIG_YAML
```

This should be sufficent to successfully deploy all components.

The rest of the section discusses managing individual components.

## Syncer

Create policynodes:

```console
$ kubectl apply -f examples/acme/policynodes/acme.yaml
```

List the created policynodes:

```console
$ kubectl get pn
```

List the corresponding namespaces:

```console
$ kubectl get ns
```

See the syncer logs:

```console
$ kubectl logs -l app=syncer --namespace nomos-system
```

## ResourceQuota Admission Controller Webhook

List resourcequota objects:

```console
$ kubectl get --all-namespaces quota
```

See the controller logs:

```console
$ kubectl logs -l app=resource-quota --namespace nomos-system
```

To test the hierarchical quota check, try creating configmaps.

This one should work:

```console
$ kubectl create configmap map1 --namespace new-prj
```

This one will be blocked thereafter due to not enough quota.

```console
$ kubectl create configmap map2 --namespace new-prj
```

## PolicyNodes Admission Controller Webhook

List policy node objects:

```console
$ kubectl get --all-namespaces pn
```

See the controller logs:

```console
$ kubectl logs -l app=policy-admission-controller -n nomos-system
```

To test the policy node check, try removing policy nodes.

Removing a child node should work:

```console
$ kubectl -n nomos-system delete pn new-prj
```

When you try and delete the root node, it should fail.

```console
$ kubectl -n nomos-system delete pn acme
```

## GitPolicyImporter

To see logs for policy-importer container:

```console
$ kubectl logs -l app=git-policy-importer -c policy-importer -n nomos-system
```

To see logs for git-sync side-car container:

```console
$ kubectl logs -l app=git-policy-importer -c git-sync -n nomos-system
```

The GitPolicyImporter is configured using a ConfigMap git-policy-importer. You
can modify that configuration and scale down the deployment and back up again to
try configuration changes.

```console
$ kubectl edit configmap git-policy-importer -n nomos-system
$ kubectl scale deployment git-policy-importer -n nomos-system --replicas=0
$ kubectl scale deployment git-policy-importer -n nomos-system --replicas=1
```

## GKE Policy Management kubectl plugin

This installs GKE Policy Management-specific commands as a `kubectl` plugin.

```console
$ make install-kubectl-plugin
```

After installation, this command gets the namespace hierarchy starting from the
namespace `frontend`.

```console
kubectl plugin nomos get namespaces --namespace=frontend
```

The `--as` impersonation works as well:

```console
$ kubectl plugin nomos get namespaces --namespace=frontend --as=alice@acme.com
```

This command gets all applicable GKE Policy Management quota objects.

```console
$ kubectl plugin nomos get quota --namespace=frontend
```

This command gets all roles across namespaces.

```console
$ kubectl plugin nomos get roles --namespace=frontend
```

This command gets all role bindings across namespaces.

```console
$ kubectl plugin nomos get rolebindings --namespace=frontend
```
