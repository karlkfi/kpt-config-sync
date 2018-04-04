# Syncer (Level Based)

## Summary

The syncer leverages the
[kubebuilder](https://github.com/kubernetes-sigs/kubebuilder) framework that
allows for rapid controller development by making a common framework for
translating resource events into triggers for reconciliation of a resource. The
new syncer will be responsible for handling hierarchical resources declared in
policy nodes as well as cluster level resources and follow the standard
kubernetes many controllers in one paradigm.

## Overview

Kubebuilder uses an event based system for triggering a "reconcile" where
declared/intended state is compared to actual and resources are updated. For
cluster level policies this is very straightforward, however, for hierarchical
resources, this requires translating incoming events to events for the entire
subtree of the node. On events for a workload namespace, the syncer will handle
re-calculating the declared policy and synchronizing it to the namespace.

## Component Overview

The syncer will have the following high level componets:

1.  PolicyHierarchyController - A component resposible for synchronizing
    hierarhcical policies. Specific policies will implement an interface which
    provides type-specific operations.
1.  ClusterPolicyController - A component responsible for synchronizing cluster
    level policies. Specific policies will also implement an interface for
    type-specific operations.

### PolicyHierarchyController Overview

The PolicyHierarchyController will listen on events for PolicyNodes. Since a PolicyNode represents a
node in a hierarchy, the entire subtree will need to be updated. The controller will take incoming
PolicyNode events and translate those to the affected namespace names and trigger events in the
kubebuilder framework to reconcile the namespace. Controlled policies and namespaces themselves will
only trigger based on the namespace name. When reconciling each namespace, the namespace's policies
be recomputed from the ancestry and applied to the controlled policies.

1.  ParentIndexer - An extension to the cache.SharedIndexInformer that provides
    parent to child lookup by adding additional indexes
1.  Hierarchy - A library for computing hierarchical policies given
    implementation of a type-specific accumulation function.
1.  Reconciler - A library for comparing declared resources to actual resources
    given a type-specific equivalence function.
1.  EventProcessor - An extension for kubebuilder that translates PolicyNode
    resource events into events for the subtree rooted at the policy node.

#### Policy Specific Handling

### ClusterPolicyController

The cluster policy controller will listen to the ClusterPolicy resource and unpack the declared
resources. The Reconclier from the PolicyHierarchyController will be re-used for handling
synchronization of declared resources to actual resources. Since there is no hierarchy involved, this
will be much simpler, however, each resource will need to have an owner reference in order to
properly remove resources.
