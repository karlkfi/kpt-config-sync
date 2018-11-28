# Installing GKE Policy Management

The Nomos Operator is a controller that manages an installation of GKE Policy
Management in a Kubernetes cluster. It consumes a Custom Resource Definition
(CRD) called Nomos, which specifies the paramaters of an installation of GKE
Policy Management.

Follow these instructions to install GKE Policy Management into your cluster
using the Nomos Operator.

This setup takes about 30 minutes.

### Prepare Installation Environment

Prerequisites:

*   Installed `curl`: to download the installation script.
*   Installed `bash`: to run the installation script.
*   Installed `kubectl`: for applying YAML files.

### Kubernetes

You need to have up and running Kubernetes clusters that you intend to install
GKE Policy Management on. You must be able to contact these clusters using
`kubectl` from the installation environment.

In order to run GKE Policy Management components, the cluster has to meet these
requirements:

<table>
  <tr>
   <td><strong>Requirement</strong>
   </td>
   <td><strong>kube-apiserver flag</strong>
   </td>
  </tr>
  <tr>
   <td>Enable RBAC
   </td>
   <td>Add <em>RBAC</em> to list passed to <em>--authorization-mode</em>
   </td>
  </tr>
  <tr>
   <td>Enable ResourceQuota admission controller
   </td>
   <td>Add <em>ResourceQuota</em> to list passed to <em>--admission-control</em>
   </td>
  </tr>
  <tr>
   <td>Enable ValidatingAdmissionWebhook
   </td>
   <td>Add <em>ValidatingAdmissionWebhook</em> to list passed to <em>--admission-control</em>
   </td>
  </tr>
</table>

Minimum required Kubernetes Server Version: **1.9**

Note that GKE running K8S 1.9 satisfies all these requirements.

**Warning:** In the current release of GKE Policy Management, we require that
all namespaces be managed by GKE Policy Management. It is recommended to create
a new cluster for use with GKE Policy Management.

The easiest way to get all of these is to follow the
[GKE quick start guide](https://cloud.google.com/kubernetes-engine/docs/quickstart)
and make sure to select version 1.9+ when creating the cluster.

## Installing

### Create Cluster-Admin ClusterRoleBinding

Ensure that the current user has cluster-admin in the cluster

```console
$ kubectl create clusterrolebinding cluster-admin-binding --clusterrole cluster-admin --user [CURRENT_USERNAME]
```

### Download Operator Manifest Bundle

Download the operator bundle directly to your machine

```console
$ curl -LO https://storage.googleapis.com/nomos-release/operator-latest/nomos-operator.yaml
```

### Deploy the Operator

Apply the operator bundle in order to create the Nomos Operator and nomos-system
namespace into your cluster.

```console
$ kubectl apply -f nomos-operator.yaml
```

You can verify that the Nomos Operator was deployed correctly
```console
$ kubectl -n kube-system get pods | grep nomos
nomos-operator-6f988f5fdd-4r7tr 1/1 Running 0 26s
```

and that the nomos-system namespace was created
```console
$ kubectl get ns | grep
nomos nomos-system Active 1m
```

### Create the Nomos Config File

The Nomos resource is a Kubernetes Custom Resource Definition (CRD) that defines
a Nomos installation. The `spec` field of a Nomos resources specifies the
installation parameters for Nomos.

An example config using
[foo-corp GitHub repo](https://github.com/frankfarzan/foo-corp-example/tree/0.1.0)
with SSH authentication is shown below:

```yaml
apiVersion: addons.sigs.k8s.io/v1alpha1
kind: Nomos
metadata:
  name: nomos
  namespace: nomos-system
spec:
  git:
    syncRepo: git@github.com:frankfarzan/foo-corp-example.git
    syncBranch: 0.1.0
    secretType: ssh
    policyDir: foo-corp
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
`secretType` | The type of secret configured for access to the Git repository. One of "ssh" or "cookiefile". Required.

### Create the git-creds Secret

#### Using SSH

Follow this process when `secretType` is set to `ssh`.

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
$ kubectl create secret generic git-creds -n=nomos-system \
    --from-file=ssh=$HOME/.ssh/id_rsa.nomos
```

#### Using GitCookies

Follow this process when `secretType` is set to `cookiefile`.

The process for acquiring a gitcookie depends on the configuration of your git
server your repository is on, but is commonly used as an authentication
mechanism for some hosting services, such as Google Cloud Source Repositories
and Gerrit. Git Cookies are usually stored in `$HOME/.gitcookies` on the local
machine.

Use the same secret you configured in the previous step to create the
`git-creds` secret in your cluster.

```console
$ kubectl create secret generic git-creds -n=nomos-system \
    --from-file=cookie-file=$HOME/.gitcookies
```

Note that these secrets are deployed into the nomos-system namespace, so it is
necessary to have that namespace created before creating the secret

### Create the Nomos Resource

Use the Nomos Resource yaml you created in the previous step to create the
Resource in the cluster.

```console
$ kubectl create -f nomos.yaml
```

### Verify Installation

To verify that GKE Policy Management components are correctly installed, issue
the following command and verify that all components listed have status
displayed as "Running."

Check running components:

```console
$ kubectl get pods -n=nomos-system
NAME                                                  READY     STATUS    RESTARTS   AGE
git-policy-importer-66bf6b9db4-pbsxn                  2/2       Running   0          24m
monitor-6f968db9-mc2xp                                1/1       Running   0          24m
syncer-58545bc77d-l485n                               1/1       Running   0          24m
```

## Uninstalling

To uninstall nomos from your cluster, delete the Nomos Resource

```console
$ kubectl -n=nomos-system delete nomos --all
```

The affected components are:

*   Everything inside namespace `nomos-system`, with the exception of the
    created `git-creds` secret.
*   Any cluster level roles and role bindings installed by GKE Policy
    Management.
*   Any admission controller configurations installed by GKE Policy Management.

[< Back](../../README.md)
