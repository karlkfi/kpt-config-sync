# System Overview

![drawing](../img/nomos_arch.png)

The above diagram is a simplified view of GKE Policy Management components
running on a workload cluster. Each component is described below.

## PolicyImporter

PolicyImporter is an abstraction for a controller that consumes policy
definitions from an external source of truth and builds a canonical
representation of the hierarchy using cluster-level CRD(s) defined by GKE Policy
Management. GKE Policy Management can be extended to support different sources
of truth (e.g. Git, GCS, Active Directory) using different implementations of
this abstraction. Note that we treat this canonical representation as internal
implementation which should not be directly consumed by users.

## CustomResourceDefinitions

GKE Policy Management defines three custom resources:

*   PolicyNode: A resource that stores hierarchical policy information. This
    includes Roles, RoleBindings and ResourceQuota. PolicyNodes form a tree,
    where leaf nodes represent Namespaces.
*   ClusterPolicy: A resource that stores cluster-level resources such as
    ClusterRoles and PodSecurityPolicies. There is only one ClusterPolicy per
    cluster.
*   Sync: A resource that stores the resource types that GKE Policy Management
    will sync from the source of truth.

## Syncer

A controller (currently packaged as a single binary) that consumes the canonical
representation of the hierarchy produced by PolicyImporter and performs CRUD on
namespaces and [sync-enabled](system_config.md#Sync) resources.

## ResourceQuotaAdmissionController

A ValidatingAdmissionWebhook that enforces hierarchical quota policies which
provides hierarchical quota on top of the existing ResourceQuota admission
controller. This is an optional component if the user chooses not to use
[hierarchical Resource Quota feature](rq.md).

## Monitor

Monitor is a controller that watches the ClusterPolicy and all PolicyNodes as
they get updated by the PolicyImporter and Syncer. It aggregates status such as
how many policies are synced or stale and the latency between import and sync.
All metrics are exported as Prometheus metrics and documented on the [Monitoring
page](monitoring_and_debugging.md#gke-policy-management-metrics).

## NomosOperator

NomosOperator is a [standard operator](https://coreos.com/operators/) which is
used to install Nomos components on a cluster and update them as new versions
become available.

[< Back](../../README.md)
