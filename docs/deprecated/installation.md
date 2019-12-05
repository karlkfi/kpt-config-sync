# Installing CSP Configuration Management

The Nomos Operator is a controller that manages an installation of GKE Policy
Management in a Kubernetes cluster. It consumes a Custom Resource Definition
(CRD) called Nomos, which specifies the paramaters of an installation of GKE
Policy Management.

Follow these instructions to install CSP Configuration Management into your
cluster using the Nomos Operator.

This setup takes about 30 minutes.

### Prepare Installation Environment

Prerequisites:

*   Installed `curl`: to download the installation script.
*   Installed `bash`: to run the installation script.
*   Installed `kubectl`: for applying YAML files.

### Kubernetes

You need to have up and running Kubernetes clusters that you intend to install
CSP Configuration Management on. You must be able to contact these clusters
using `kubectl` from the installation environment.

In order to run CSP Configuration Management components, the cluster has to meet
these requirements:

Requirement                               | kube-apiserver flag
----------------------------------------- | -------------------
Enable RBAC                               | Add `RBAC` to list passed to `--authorization-mode`
Enable ValidatingAdmissionWebhook         | Add `ValidatingAdmissionWebhook` to list passed to `--admission-control`

Minimum required Kubernetes Server Version: **1.10**

Note that GKE running K8S 1.10 satisfies all these requirements.

**Warning:** In the current release of CSP Configuration Management, we require
that all namespaces be managed by CSP Configuration Management. It is
recommended to create a new cluster for use with CSP Configuration Management.

The easiest way to get all of these is to follow the
[GKE quick start guide](https://cloud.google.com/kubernetes-engine/docs/quickstart).

## Installing

### Create Cluster-Admin ClusterRoleBinding

Ensure that the current user has cluster-admin in the cluster:

```console
$ kubectl create clusterrolebinding cluster-admin-binding --clusterrole cluster-admin --user <user>
```

On GKE clusters, `<user>` would be the GSuite account (e.g.
`charlie@foo-corp.com`).

### Deploy the Operator

Apply the operator bundle in order to create the Nomos Operator and
config-management-system namespace into your cluster.

```console
$ kubectl apply --filename https://storage.googleapis.com/nomos-release/operator-stable/config-management-operator.yaml
```

You can verify that the Nomos Operator was deployed correctly:

```console
$ kubectl -n kube-system get pods | grep nomos
nomos-operator-6f988f5fdd-4r7tr 1/1 Running 0 26s
```

and that the config-management-system namespace was created:

```console
$ kubectl get ns | grep nomos
config-management-system Active 1m
```

### Create the git-creds Secret

Note that these secrets are deployed into the config-management-system
namespace, so it is necessary to have that namespace created before creating the
secret.

Choose the correct authentication method for your Git repository from the
options below. The method chosen will determine the value you use for
`secretType` when creating the `Nomos` resource in the next section.

#### Using `ssh`

Follow this process when `secretType` is set to `ssh`. This would be use when
authenticating to a Git repo using ssh keys.

First, create a nomos-specific private/public key pair.

```console
$ ssh-keygen -t rsa -b 4096 -C "alice@example.com" -N '' -f $HOME/.ssh/id_rsa.nomos
```

Whether to use a different key per cluster is up to the needs of your system.

Next, configure the server where your repository is hosted to recognize the
newly created public key, `id_rsa.nomos.pub`. This process depends on how your
repository is hosted. For an example: if your repository is hosted on Github,
follow
[this process](https://help.github.com/articles/adding-a-new-ssh-key-to-your-github-account/)
to add the public key to your Github account.

Use the same secret you configured in the previous step to create the
`git-creds` secret in your cluster.

```console
$ kubectl create secret generic git-creds -n=config-management-system \
    --from-file=ssh=$HOME/.ssh/id_rsa.nomos
```

#### Using `cookiefile`

Follow this process when `secretType` is set to `cookiefile`. This would be used
when authenticating to a Git repo using Git cookies.

The process for acquiring a Git cookie depends on the configuration of your Git
server your repository is on, but is commonly used as an authentication
mechanism for some hosting services, such as Google Cloud Source Repositories
and Gerrit. Git Cookies are usually stored in `$HOME/.gitcookies` on the local
machine.

Use the same secret you configured in the previous step to create the
`git-creds` secret in your cluster.

```console
$ kubectl create secret generic git-creds -n=config-management-system \
    --from-file=cookie_file=$HOME/.gitcookies
```

#### Using `none`

No secret is needed when `secretType` is set to `none`. This would be used when
accessing a repo via a public https:// target, which implies the repo is open to
the public. __Such a configuration is not recommended.__

### Create the Nomos Resource

The Nomos resource is a Kubernetes Custom Resource (CR) that defines a Nomos
installation on a given cluster. The `spec` field of a Nomos resource specifies
the installation parameters for Nomos.

An example config using
[foo-corp GitHub repo](https://github.com/frankfarzan/foo-corp-example/tree/0.1.0)
with SSH authentication is shown below. One Nomos resource is needed per
cluster.

```yaml
apiVersion: addons.sigs.k8s.io/v1alpha1
kind: Nomos
metadata:
  name: nomos
  namespace: config-management-system
spec:
  clusterName: some-cluster-name
  git:
    syncRepo: git@github.com:frankfarzan/foo-corp-example.git
    syncBranch: 0.1.0
    secretType: ssh
    policyDir: foo-corp
  enableHierarchicalResourceQuota: true
```

`spec` contains a top level field `git`, which is an object with the following
properties:

Key          | Description
------------ | -----------
`syncRepo`   | The URL of the Git repository to use as the source of truth. Required.
`syncBranch` | The branch of the repository to sync from. Default: master.
`policyDir`  | The path within the repository to the top of the policy hierarchy to sync. Default: the root directory of the repository.
`syncWait`   | Period in seconds between consecutive syncs. Default: 15.
`syncRev`    | Git revision (tag or hash) to check out. Default HEAD.
`secretType` | The type of secret configured for access to the Git repository. One of "ssh", "cookiefile" or "none". Required.

In addition, the following top level properties can be specified:

Key                               | Description
--------------------------------- | -----------
`clusterName`                     | User-defined name for the cluster used by [ClusterSelectors](clusterselectors.md) to group clusters together. Unique in a Nomos installation.

Note: Hierarchical Resource Quota is currently unsupported on GKE On Prem Alpha
and the flag should be set to false or omitted.

Once you have created your Nomos resource file, apply it to the API server.
Repeat the step for each cluster in an installation.

```console
$ kubectl apply -f nomos.yaml
```

You may need to apply different Nomos resources to different clusters.

```console
$ kubectl apply -f nomos1.yaml --context=cluster-1
$ kubectl apply -f nomos2.yaml --context=cluster-2
```

### Verify Installation

To verify that CSP Configuration Management components are correctly installed,
issue the following command and verify that all deployments listed have status
displayed as "Running."

Check running components:

```console
$ kubectl get pods -n=config-management-system
```

If the above components do not appear, you may find relevant error messages in
the operator logs:

```console
kubectl -n kube-system logs -l k8s-app=nomos-operator
```

## Uninstalling

To uninstall nomos from your cluster, delete the Nomos Resource:

```console
$ kubectl -n=config-management-system delete nomos --all
```

The affected components are:

*   Everything inside namespace `config-management-system`, with the exception
    of the created `git-creds` secret.
*   Any cluster level roles and role bindings installed by GKE Policy
    Management.
*   Any admission controller configurations installed by CSP Configuration
    Management.

### Uninstalling the Operator

Usually, uninstalling Nomos's functional components with the instructions above
should be sufficient and it is not necessary to uninstall the Operator. The
Operator can safely remain on the cluster to assist with future reinstallation,
and will not affect the cluster until the Nomos CRD is created. If the Operator
is removed, it will have to be re-created before Nomos can be installed again.
Users of GKE on-prem should not attempt to uninstall the operator, as it is
managed by the GKE on-prem platform.

However, if you do wish to completely remove the Operator and associated
resources, follow this process:

First uninstall Nomos as [above](#uninstalling), and wait until the
`config-management-system` namespace is empty.

Make sure that `config-management-system` has no resources before proceeding:

```console
$ kubectl -n config-management-system get all
No resources found.
```

Delete the `config-management-system` namespace:

```console
$ kubectl delete ns config-management-system
```

Then, delete the Nomos CRD:

```console
$ kubectl delete crd nomos.addons.sigs.k8s.io
```

And delete all `kube-system` resources for the operator:

```console
$ kubectl -n kube-system delete all -l k8s-app=nomos-operator
```

Note: there is one additional CRD called `applications.app.k8s.io` that is
installed with the Nomos Operator bundle, *but* this CRD is also used by some
other GKE
[add-ons](https://kubernetes.io/docs/concepts/cluster-administration/addons/).
If you are confident that no other add-ons are using this CRD, it can be deleted
as well.

[< Back](../../README.md)
