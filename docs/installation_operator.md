# Installing GKE Policy Management (Operator Install)

The Nomos Operator is a controller that manages an installation of GKE Policy Management in a Kubernetes cluster. It consumes a Custom Resource Definition (CRD) called Nomos, which specifies the paramaters of an installation of GKE Policy Management.

Follow these instructions to install GKE Policy Management into your cluster using the Nomos Operator.

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

```$bash
$ kubectl create clusterrolebinding cluster-admin-binding --clusterrole cluster-admin --user [CURRENT_USERNAME]
```

### Download Operator Manifest Bundle
Download the operator bundle directly to your machine

```$bash
$ curl -LO https://storage.googleapis.com/nomos-release/operator-latest/nomos-operator.yaml
```

### Deploy the Operator
Apply the operator bundle in order to create the Nomos Operator and nomos-system namespace into your cluster.

```$bash
$ kubectl apply -f nomos-operator.yaml
```

You can verify that the Nomos Operator was deployed correctly
```$bash
$ kubectl -n kube-system get pods | grep nomos
nomos-operator-6f988f5fdd-4r7tr       1/1       Running   0          26s
```

and that the nomos-system namespace was created
```$bash
$ kubectl get ns | grep nomos
nomos-system   Active    1m
```

### Prepare Nomos Resource and Secret File
Follow the [Operator Config Guide](nomos_config.md) to write a Nomos resource specifying the parameters of your desired installation and create the secret file for your installation.

### Create the git-creds Secret

Use the same secret you configured in the previous step to create the `git-creds` secret in your cluster.

For example, create the ssh secret with:

```console
$ kubectl create secret generic git-creds -n=nomos-system \
    --from-file=ssh=$HOME/.ssh/id_rsa.nomos
```

or, create the git-cookies secret with:

```console
$ kubectl create secret generic git-creds -n=nomos-system \
    --from-file=cookie-file=~/.gitcookie
```

Note that these secrets are deployed into the nomos-system namespace, so it is necessary to have that namespace created before creating the secret

### Create the Nomos Resource

Use the Nomos Resource yaml you created in the previous step to create the Resource in the cluster.

```$bash
kubectl create -f nomos.yaml
```

### Verify Installation

To verify that GKE Policy Management components are correctly installed, issue
the following command and verify that all components listed have status
displayed as "Running."

Check running components:

```console
$ kubectl get pods -n=nomos-system
NAME                                                  READY     STATUS    RESTARTS   AGE
git-policy-importer-66bf6b9db4-pbsxn            2/2       Running   0          24m
monitor-6f968db9-mc2xp                                1/1       Running   0          24m
policy-admission-controller-6746f96cbb-2h2sf          1/1       Running   0          24m
resourcequota-admission-controller-64988d97f4-nxmsc*  1/1       Running   0          24m
syncer-58545bc77d-l485n                               1/1       Running   0          24m
```

## Uninstalling

To uninstall nomos from your cluster, delete the Nomos Resource

```$bash
kubectl -n=nomos-system delete nomos --all
```

The affected components are:

*   Everything inside namespace `nomos-system`, with the exception of the created `git-creds` secret.
*   Any cluster level roles and role bindings installed by GKE Policy
    Management.
*   Any admission controller configurations installed by GKE Policy Management.
