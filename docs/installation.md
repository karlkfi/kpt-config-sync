# Installing Nomos

## Requirements

Before installing Nomos, there are a few tasks that must be performed once to
ensure that your work environment is complete and able to support the
installation.

This setup takes about 30 minutes.

### Installation environments

The installation environments are sessions on a computer that you will run the
installation from. Any environments not mentioned here explicitly are untested,
so we can not make guarantees about them.

#### Linux

Currently the only tested environment for installation is Ubuntu 14.04 on amd64.

Prerequisites:

*   Installed `curl`: to download the installation script.
*   Installed `docker`: to run the installation script.
*   Installed `bash`: to run the installation script.

### Kubernetes

You need to have up and running Kubernetes clusters that you intend to install
Nomos on. You must be able to contact these clusters using `kubectl` from the
installation environment.

In order to run Nomos components, the cluster has to meet these requirements:

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

**Warning:** In the current release of Nomos, we require that all namespaces be
managed by Nomos. It is recommended to create a new cluster for use with Nomos.

The easiest way to get all of these is to follow the
[GKE quick start guide](https://cloud.google.com/kubernetes-engine/docs/quickstart)
and make sure to select version 1.9+ when creating the cluster.

## Installation

Download the Nomos installer script to a directory on your machine.

```console
$ cd
$ mkdir -p tmp/nomos
$ cd tmp/nomos
$ curl https://storage.googleapis.com/nomos-release/stable/run-installer.sh -o run-installer.sh
$ chmod +x run-installer.sh
```

Installation configuration is defined in a YAML file. When using Git, follow the
guide for [Git Configuration](git_config.md) to create this config file. When
using GCP, follow the guide for [GCP Configuration](gcp_config.md) instead.

Once you have created a config file, you can run the installer as follows:

```console
$ ./run-installer.sh --config=/path/to/your/config.yaml
```

## Verify installation

To verify that Nomos components are correctly installed, issue the following
command and verify that all components listed have status displayed as
"Running."

Check running components:

```console
$ kubectl get pods -n=nomos-system
NAME                                                  READY     STATUS    RESTARTS   AGE
[git|gcp]-policy-importer-66bf6b9db4-pbsxn            2/2       Running   0          24m
policy-admission-controller-6746f96cbb-2h2sf          1/1       Running   0          24m
resourcequota-admission-controller-64988d97f4-nxmsc   1/1       Running   0          24m
syncer-58545bc77d-l485n                               1/1       Running   0          24m
```

## Uninstalling

To uninstall Nomos from a set of clusters, you need the `config.yaml` file used
for the original installation, and the `run-installer.sh` script.

Executing the following command will uninstall Nomos components.

```console
./run-installer.sh --config=/path/to/your/config.yaml --uninstall=deletedeletedelete

```

The affected components are:

*   The namespace `nomos-system` along any workloads running inside of it.
*   Any cluster level roles and role bindings installed by Nomos.
*   Any admission controller configurations installed by Nomos.

In addition, removing Nomos from the cluster may affect user workloads that
interact with the Kubernetes API server.

## Reinstalling

It is possible to reuse an existing installer configuration multiple times to
reinstall Nomos. To run the reinstall use the batch installation mode with your
existing configuration:

```console
$ ./run-installer.sh --config=/path/to/your/config.yaml
```

The effect of the reinstallation is to run the equivalent of `kubectl apply` to
almost all the Kubernetes components in the Nomos installation package. The
exception are certificates and required secrets, which are removed prior to the
bulk of reinstall process. This has the effect of installing a fresh copy of the
certificates and secrets. Namespaces and deployments may not be affected if the
reinstall would not change their state.
