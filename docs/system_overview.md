# System Overview

![drawing](img/nomos_arch.png)

The above diagram is a simplified view of Nomos components running on a workload
cluster. Each component is described below.

## PolicyImporter

PolicyImporter is an abstraction for a controller that consumes policy
definitions from an external source of truth and builds a canonical
representation of the hierarchy using cluster-level CRD(s) defined by Nomos.
Nomos can be extended to support different sources of truth (e.g. Git, GCP,
Active Directory) using different implementations of this abstraction. Note that
we treat this canonical representation as internal implementation which should
not be directly consumed by users.

## Syncer

A set of controllers (currently packaged as a single binary) that consume the
canonical representation of the hierarchy produced by PolicyImporter and perform
CRUD on namespaces and native K8S policy resources such as Role/Rolebinding,
ResourceQuota, etc.

## ResourceQuotaAdmissionController

A ValidatingAdmissionWebhook that enforces hierarchical quota policies which
providers hierarchical quota on top of existing ResourceQuota admission
controller.
