# Setup

## Depenendencies

1.  [Helm](https://docs.helm.sh/using_helm/#installing-helm)
    1.  Go to the [Helm releases](https://github.com/kubernetes/helm/releases)
        page, find the heading "Installation and Upgrades", and download the
        binary archive for your OS of choice, e.g. likely Linux for Nomos work.
    1.  From the downloaded archive in the previous step, extract the `helm`
        binary to somewhere in your path.
    1.  Run `helm init --client-only`. This makes the next step possible.
    1.  Add necessary helm repos: `helm repo add coreos
        https://s3-eu-west-1.amazonaws.com/coreos-charts/stable/`
1.  [go 1.10](https://golang.org/doc/install)
    1.  You should install it under your home directory (eg `$HOME/opt/go`).
    1.  Set `$GOROOT` to this directory (eg `export GOROOT=$HOME/opt/go`).
    1.  Set `$GOPATH` to a different directory under home (eg `export
        GOPATH=$HOME/go`).
    1.  Add `$GOROOT` and `$GOPATH` to your path (eg `export
        PATH=$GOROOT/bin:$PATH:$GOPATH/bin`).

## Initial setup of your cluster in GCE (one time)

Configure gcloud authn credentials if you have not already done so:

```shell
gcloud auth login
```

Configure gcloud to use our shared dev project where we bring up k8s clusters:

```shell
gcloud config set project stolos-dev
gcloud config set compute/zone us-central1-b
```

Make sure you are in the correct directory:

```shell
cd $NOMOS/scripts/cluster/gce
```

Download kubernetes (1 min).

```shell
./download-k8s-release.sh
```

Start up your cluster (10 min):

```shell
./kube-up.sh
```

Set up prometheus for monitoring:

```shell
./configure-monitoring.sh
```

Ensure you are properly connected to the cluster:

```shell
kubectl get ns # lists the 3 default namespaces plus monitoring namespace
```

## Initial setup of your cluster in GKE (one time)

Create cluster using gcloud

```shell
export CLUSTER_NAME=<yourclustername>
gcloud container clusters create ${CLUSTER_NAME} \
--enable-autoupgrade \
--machine-type=f1-micro \
--zone=us-central1-a \
--num-nodes=3 \
--cluster-version=1.9.4-gke.0
```

OR Optionally Create a GKE Cluster with version 1.9.4 or above and connect to it
(Pantheon).

Set up prometheus for monitoring:

```shell
$NOMOS/scripts/cluster/gce/configure-monitoring.sh
```

Add yourself as a cluster admin in RBAC. See
https://cloud.google.com/kubernetes-engine/docs/how-to/role-based-access-control
for details of why this is needed.

```shell
kubectl create clusterrolebinding {{USERNAME}}-cluster-admin-binding \
    --clusterrole=cluster-admin --user={{USERNAME}}@google.com
```
