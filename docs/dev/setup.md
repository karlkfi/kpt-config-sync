# Setup

## Dependencies

1.  [go 1.10](https://golang.org/doc/install)
    1.  You should install it under your home directory (eg `$HOME/opt/go`).
    1.  Set `$GOROOT` to this directory (eg `export GOROOT=$HOME/opt/go`).
    1.  Set `$GOPATH` to a different directory under home (eg `export
        GOPATH=$HOME/go`).
    1.  Add `$GOROOT` and `$GOPATH` to your path (eg `export
        PATH=$GOROOT/bin:$PATH:$GOPATH/bin`).

## Initial setup of your cluster in GCE (one time)

Configure gcloud authn credentials if you have not already done so:

```console
$ gcloud auth login
```

Configure gcloud to use our shared dev project where we bring up k8s clusters:

```console
$ gcloud config set project stolos-dev
$ gcloud config set compute/zone us-central1-b
```

Make sure you are in the correct directory:

```console
$ cd $NOMOS/scripts/cluster/gce
```

Download kubernetes (1 min).

```console
$ ./download-k8s-release.sh
```

Start up your cluster (10 min):

```console
$ ./kube-up.sh
```

Ensure you are properly connected to the cluster:

```console
$ kubectl get ns # lists the 3 default namespaces
```

## Initial setup of your cluster in GKE (one time)

Create a cluster using the [web UI](https://console.cloud.google.com) or the
[gcloud CLI](https://cloud.google.com/sdk/gcloud/reference/container/clusters/create).
Make sure the cluster version is 1.9+ (we suggest using the latest version). You
should also select a machine type that has at least 2GB of RAM (eg
n1-standard-1).

## Configure monitoring (optional)

You can now configure the server side processes for monitoring GKE Policy 
Management. This step is optional but we strongly recommend it since the 
monitoring can be useful for debugging as well as measuring performance.

Install [Helm](https://docs.helm.sh/using_helm/#installing-helm):

1.  Go to the [Helm releases](https://github.com/kubernetes/helm/releases) page,
    find the heading "Installation and Upgrades", and download the binary
    archive for your OS of choice, e.g. likely Linux for GKE Policy Management 
	work.
1.  From the downloaded archive in the previous step, extract the `helm` binary
    to somewhere in your path.
1.  Run `helm init --client-only`. This makes the next step possible.
1.  Add necessary helm repos: `helm repo add coreos
    https://s3-eu-west-1.amazonaws.com/coreos-charts/stable/`

```console
$ $NOMOS/scripts/cluster/gce/configure-monitoring.sh
```

You can verify that there are now several pods running in the new monitoring
namespace:

```console
$ kubectl get pods -n monitoring
```
