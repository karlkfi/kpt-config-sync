# Nolos

Nolos enables policy distribution and hierarchical policy enforcement for
multi-tenant multi-cluster Kubernetes deployments.

See [user guide](docs/user_guide.md).

## Overview {#overview}

In environments with many users spread across many teams, having multiple
tenants within a cluster, allocated into namespaces, maximizes resource
utilization while providing isolation. However, as the number of namespaces
grows, it becomes increasingly hard for cluster operators to manage
per-namespace policies like authorization (RBAC Roles/Rolebinding) and quota
(ResourceQuota), etc. In addition, real world deployments often require multiple
clusters in order to tolerate region failures, reduce network latencies for end
users, or simply to scale beyond the current size limits of a Kubernetes
cluster. Nolos makes it easier to manage large multi-tenant and multi-cluster
deployments, by reducing the load on cluster operators and reducing the surface
area to secure.

At a high level, Nolos serves two separate functions:

*   **Policy distribution**: Distribute policy definitions from a centralized
    source of truth to all workload clusters. The extensible policy distribution
    mechanism allows Nolos to support a wide range of technologies implementing
    the source of truth for policies (e.g. YAML files in Git, Google Cloud IAM &
    admin, Active Directory, etc.)
*   **Hierarchical policy enforcement**: Group together namespaces and
    associated policies into a hierarchy modelled after how departments and
    teams are organized. Nolos provides a set of controllers running on the
    workload clusters that consume hierarchical policies definitions and are
    responsible for enforcing them.

In this release, there are four main areas that Nolos helps manage:

1.  **Namespaces**: With Nolos you have one set of namespaces that apply to all
    clusters. Nolos also introduces hierarchical namespace support giving you
    the ability to group namespaces together for common policies and to
    facilitate delegation. In Nolos, only leaf namespaces can contain
    non-policy resources, while the intermediate and root nodes provide policy
    attach points for policies such as RBAC, ResourceQuota, and more.
1.  **Hierarchical RBAC policies**: Nolos provides central management of RBAC
    policies and enables inheritance of namespace-level RBAC resources. For
    example a Rolebinding from an ancestor is inherited by all descendent
    namespaces, removing duplication.
1.  **Hierarchical ResourceQuota policies**: With Nolos one can manage quota
    centrally and set quota hierarchically.
1.  **Cluster-level policies:** In addition to namespace-level policies, Nolos
    allows you to centrally manage cluster-level policies such as
    ClusterRole/Rolebinding and PodSecurityPolicy.

This guide will take you through managing each of these resources.

## Nolos Concepts {#nolos-concepts}

### Glossary {#glossary}

*   **Workload cluster:** A cluster where a user runs a pod.
*   **Enrolled cluster:** A workload cluster running Nolos components.

### Namespaces {#namespaces}

In Kubernetes, namespaces are the isolation construct for implementing
multi-tenancy and are the parents of all workload resources (pods, replica sets,
service accounts, etc). Namespaces can be used to isolate workloads that have
the same set of human owners as well as to isolate different workload
environments.

Generally anytime you want to have a workload managed by a distinct person or
set of people (e.g. on-call person only for prod workloads, whole development
team for dev workloads), it makes sense to create a new namespace. If you want
to have a common person or set of people be able to perform the same set of
operations within a set of namespaces, create a policyspace.

Nolos with its hierarchical control allows many namespaces to be managed by the
same set of people so it's possible to create more granular namespaces for a
team of people without incurring additional policy administration overhead.

### Policyspaces {#policyspaces}

With Nolos, we give admins the ability to group namespaces together and to form
groups of groups through a hierarchy. We call a non-leaf node in this tree,
whose leaves are namespaces, a policyspace. We can think of policyspaces as
Organization Units (or Policy Information Point in
[XACML](https://en.wikipedia.org/wiki/XACML#Terminology) parlance). They exist
to delegate policy control to sub organization leaders. This approach has been
long established by LDAP and Active Directory.

Policyspaces can be parents of policyspaces and namespaces. Policyspace and
namespaces must both be globally unique.

### Delegation {#delegation}

As the organization complexity grows, it is important for an admin to have the
ability to delegate administration of a subset of the hierarchy to another
admin. The mechanism for delegation is specific to the source of truth. For
example, on Google Cloud Platform, an admin can grant setIamPolicy permission to
someone who can then set policies independently. In Git, the admin can give
commit permissions to a subtree to someone else.

### Example {#example}

One can imagine a brick and mortar retailer having a hierarchy of policyspaces
and namespaces that looks like this

Figure 1: foo-corp organization

In Foo corp, a small team (8-10 people) runs a few microservice workloads that
together provide a bigger component. Everyone on the team has the same set of
roles (e.g. deployment, on-call, coding, etc). We also assume that the small
team will run a dev and staging setup for qualification and will want to ensure
that these environments have different security postures. Also these workloads
need to run in multiple regions.

This gives the ability for the Shipping App Backend team to manage three
different namespaces but only have to maintain one authorization policy for team
members. Each of their namespaces is isolated by environment, allowing
identically-named objects in the three envionments' instantiations of the
backend stack, as well as providing tighter security, e.g. allowing one
namespace to have additional authorized users but not the others, and allocating
private quota to each namespace.

## System Overview {#system-overview}

Each component is described below.

### PolicyImporter {#policyimporter}

PolicyImporter is an abstraction for a controller that consumes policy
definitions from an external source of truth and builds a canonical
representation of the hierarchy using cluster-level CRD(s) defined by Nolos.
Nolos can be extended to support different sources of truth (e.g. Git, GCP,
Active Directory) using different implementations of this abstraction. Note that
we treat this canonical representation as internal implementation which should
not be directly consumed by users.

### Syncer {#syncer}

A set of controllers (currently packaged as a single binary) that consume the
canonical representation of the hierarchy produced by PolicyImporter and perform
CRUD on namespaces and native K8S policy resources such as Role/Rolebinding,
ResourceQuota, etc.

### ResourceQuotaAdmissionController {#resourcequotaadmissioncontroller}

A ValidatingAdmissionWebhook that enforces hierarchical quota policies which
providers hierarchical quota on top of existing ResourceQuota admission
controller.

# Quickstart Set Up {#set-up}

There's a one-time set up required to set up Nolos components described in this
section. The user running these commands should have a cluster-admin
rolebinding.

## Before you begin

This is a list of tasks that must be performed once to ensure that your work
environment is complete and able to support the installation.

The setup takes about 30 minutes.

### Linux

You will need a Linux environment to drive the installation.

At the moment, the only tested environment for installation is Linux on amd64.
The following utilities need to be installed: `docker`, `bash`, `curl`, and
`gcloud`.

### Kubernetes

You will need to have at least one up-and-running Kubernetes cluster, version
1.9 or above, and have credentials available to access this cluster.

The easiest way to get all of these is to follow the [GKE quick start
guide](https://cloud.google.com/kubernetes-engine/docs/quickstart).

### SSH

You will need to create SSH credentials to access the Github sample git
repository, and ensure that those credentials are usable for github access.
IMPORTANT NOTE: You will be copying this key to a nomos service on the
kubernetes cluster in a few minutes, so you likely want to create a separate key
for this purpose.

In a terminal session of your Linux machine issue the following command:

```
ssh-keygen -N '' -f $HOME/.ssh/id_rsa.nomos
```

This command will create a pair of keys, `$HOME/.ssh.id_rsa.nomos`, and
`$HOME/.ssh.id_rsa.nomos.pub` that will be used to set up git repo access in
Nomos.

Note that the resulting private key must _not_ be password protected. It is
advisable not to reuse this key for anything else but Nomos access in this
example.

### Github

You will need an account on [github.com](http://github.com).

This account is used to access our example repositories, and to provide a source
of truth for policy synchronization. You will be able to retarget the setup to
your own repository later, if you so choose.

An example policy hierarchy is available in [this Github
repository](https://github.com/frankfarzan/foo-corp-example).

[Upload](https://help.github.com/articles/adding-a-new-ssh-key-to-your-github-account/)
the file `$HOME/.ssh/id_rsa.nomos.pub`, which was generated in the previous
step, to your account on Github. This file will be used by the Nomos
installation to access the sample git repository. The file
`$HOME/.ssh/id_rsa.nomos` should be guarded carefully as any other private key
file.

Now, [test](https://help.github.com/articles/testing-your-ssh-connection/) your
SSH connection to github using the key you just generated.

```
$ ssh -F /dev/null -i $HOME/.ssh/id_rsa.nomos -T git@github.com
Hi <your_username>! You've successfully authenticated, but GitHub does not provide shell access.
```

[Fork this repo on Github](https://help.github.com/articles/fork-a-repo/) or to
your preferred Git hosting provider if you want to make changes.

You can now clone the sample repository locally as follows:

```
$ ssh-add $HOME/.ssh/id_rsa.nomos
$ git clone git@github.com:frankfarzan/foo-corp-example.git foo
```

or your own copy as:

```
$ git clone git@github.com:your_github_username/foo-corp-example.git
```

### Installer

Download the Nomos installer script to a directory on your machine.

```
cd
mkdir -p tmp/nomos
cd tmp/nomos
curl https://storage.googleapis.com/nomos-release/run-installer.sh \
> run-installer.sh
chmod +x run-installer.sh
```

### Interactive installation

Interactive installation is menu driven. It allows you to edit the configuration
through a user-friendly menu as shown in the figure below.

You can start from a sample configuration:

```
env INTERACTIVE=1 ./run-installer.sh
```

"Save" will store the current settings that you can then reuse later in a batch
installation. "Install" will run the installer on the chosen clusters.

### Verify installation

The install verification will eventually be automated. Until that time, please
verify manually that all components that are running are in fact running
correctly.

To verify that Nomos components are installed, issue the following command and
verify that all components listed have status displayed as Running.

Check running components:

```
$ kubectl get pods -n=nolos-system
NAME                                                  READY     STATUS    RESTARTS   AGE
git-policy-importer-66bf6b9db4-pbsxn                  2/2       Running   0          24m
resourcequota-admission-controller-64988d97f4-nxmsc   1/1       Running   0          24m
syncer-58545bc77d-l485n                               1/1       Running   0          24m
```

Check created secrets:

```
$ kubectl get secrets -n nolos-system
NAME                                             TYPE                                  DATA      AGE
default-token-tjc2f                              kubernetes.io/service-account-token   3         2d
git-creds                                        Opaque                                2         2d
policy-importer-token-gwwzz                      kubernetes.io/service-account-token   3         2d
resourcequota-admission-controller-secret        kubernetes.io/tls                     2         2d
resourcequota-admission-controller-secret-ca     kubernetes.io/tls                     2         2d
resourcequota-admission-controller-token-tqw2t   kubernetes.io/service-account-token   3         2d
nolosresourcequota-controller-token-4xrw8       kubernetes.io/service-account-token   3         2d
syncer-token-kfkm7                               kubernetes.io/service-account-token   3         2d

```

Check present namespaces:

```
$ kubectl get ns
NAME               STATUS    AGE
audit              Active    2m
default            Active    2m
kube-public        Active    2m
kube-system        Active    2m
shipping-dev       Active    2m
shipping-prod      Active    2m
shipping-staging   Active    2m
nolos-system       Active    2m

```

