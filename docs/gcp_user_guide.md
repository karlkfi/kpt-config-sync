# GCP User Guide

**NOTE: GCP support is not yet available.**

Nomos supports using Google Cloud Platform to centrally manage Namespaces and
policies across Kubernetes clusters.

[Resource Manager][1] API enables creating a hierarchy using Org, Folders, and
Projects. Kubernetes Policy Namespaces API enables creating a single *Managed
Namespace* in a Project, such that there is a 1-to-1 mapping between a Project
and a Managed Namespace. The user can then set policies such as [IAM
authorization][2] on each node in the hierarchy.

Nomos automatically creates and manages corresponding Kubernetes Namespaces in
all enrolled clusters and creates Kubernetes policy resources such as RBAC based
on hierarchical evaluation of policies defined in GCP. In addition, Nomos
creates certain cluster-level resources such as ClusterRoles based on predefined
IAM roles described [here][3].

[1]: https://cloud.google.com/resource-manager
[2]: https://cloud.google.com/iam
[3]: https://cloud.google.com/kubernetes-engine/docs/how-to/iam#predefined

## Policy Hierarchy Operations

TODO(110722449)

### Creation

### Deletion

### Renaming

### Moving

## Policy Types

### Namespace-level Policies

##### Role/Rolebinding

### Cluster-level Policies

##### ClusterRole/ClusterRoleBinding
