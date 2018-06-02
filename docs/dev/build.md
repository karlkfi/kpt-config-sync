# Building

To build and deploy all Nomos binaries:

First time to create the deployment configuration:

```console
 $ cd $NOMOS
 $ make deploy-interactive
```

The interactive installer that is invoked by this command will save the
generated configuration into
`$NOMOS/.output/staging/installer_output/gen_configs/generated.yaml`.

Subsequent deployments can reuse the generated configuration. Be sure to put it
into a safe place outside of the ephemeral `$NOMOS/.output` directory if you
need to save it for later.

```console
$ cd $NOMOS
$ make NOMOS_INSTALLER_CONFIG=path/to/config deploy
```

This should be sufficent to successfully deploy all components.

Rest of the section discusses individual components.

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

## Nomos kubectl plugin

This installs Nomos-specific commands as a `kubectl` plugin.

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

This command gets all applicable Nomos quota objects.

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
