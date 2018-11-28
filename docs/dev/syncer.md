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

1.  SyncerController - Boilerplate for setup of kubebuilder controllers and
    worker queue processing. This handles setting up the other controllers
    within the kubebuilder framework then processing the generated actions.
1.  PolicyHierarchyController - A component resposible for synchronizing
    hierarhcical policies. Specific policies will implement an interface which
    provides type-specific operations.
1.  ClusterPolicyController - A component responsible for synchronizing cluster
    level policies. Specific policies will also implement an interface for
    type-specific operations.

### PolicyHierarchyController Overview

The PolicyHierarchyController will listen on events for PolicyNodes. Since a
PolicyNode represents a node in a hierarchy, the entire subtree will need to be
updated. The controller will take incoming PolicyNode events and translate those
to the affected namespace names and trigger events in the kubebuilder framework
to reconcile the namespace. Controlled policies and namespaces themselves will
only trigger based on the namespace name. When reconciling each namespace, the
namespace's policies be recomputed from the ancestry and applied to the
controlled policies.

1.  ParentIndexer - An extension to the cache.SharedIndexInformer that provides
    parent to child lookup by adding additional indexes
1.  Hierarchy - A library for computing hierarchical policies given
    implementation of a type-specific accumulation function.
1.  Reconciler - A library for comparing declared resources to actual resources
    given a type-specific equivalence function.
1.  EventProcessor - An extension for kubebuilder that translates PolicyNode
    resource events into events for the subtree rooted at the policy node.
1.  Module - An interface that handles controlling a resource from the
    definitions in the policy node.

#### Modules (Policy Specific Handling)

Each Module is responsible for controlling a single resource (eg, Role,
RoleBinding). In order to satisfy the Module interface, it must implement
methods to compute hierarchical policies from an ancestry of PolicyNode objects
as well as define a type-specific equality function which will be used for
comparing intended to actual state.

### ClusterPolicyController

The cluster policy controller will listen to the ClusterPolicy resource and
unpack the declared resources. The Reconclier from the PolicyHierarchyController
will be re-used for handling synchronization of declared resources to actual
resources. Since there is no hierarchy involved, this will be much simpler,
however, each resource will need to have an owner reference in order to properly
remove resources.

## Life of an Event

### Life of a Kubebuilder Controller

Kubebuilder provides a common library for performing reconciliation between a
declaring resource and a controlled resource. It entails setting up an informer
on both the declared resource as well as the controlled resource for the purpose
of gathering events. On each event, a reconcile will be triggered which looks at
the declared resources and controlled resources and reconciles the difference
between the two. Since we have created the PolicyNode resource which controls
resources on the namespace level, our granularity for a reconcile will be the
namespace and it will entail recalculating and updating the namespace as a
whole.

#### Debouncing

Since there can be a large number of events that trigger a reconciliation, the
framework provides a mechanism to debounce the incoming events. This means that
a reconcilliation will clear all events that triggered it rather than performing
the reconciliation for each and every event.

### Kubebuilder Syncer Lifecycle

1.  Pod created
1.  Syncer sets up kubebuilder framework
1.  Syncer registers GenericController for PolicyNode and ClusterPolicy with the
    ControllerManager
1.  Syncer hands control to kubebuilder framework

### PolicyHierarchy controller

1.  Controller is created with hierarchical modules.
1.  Controller registers informers for PolicyNode, Namespace, and module
    specific resources
1.  Controller registers custom hierarchical event propagating watch on
    PolicyNode via WatchEvents
1.  Controller registers Namespace events via Watch
1.  Controller registers module specific resource (eg Role) events via
    WatchTransformationOf using a function that translates resource namespace to
    policy node name.
1.  Controller returns kubebuilder GenericController for registration with
    kubebuilder ControllerManager.

### Life of an update to declared state (PolicyNodes)

1.  A PolicyNode is mutated
1.  The kubebuilder framework recieves the event and hands it to the
    hierarchical handler
1.  The hierarchical handler produces events for all PolicyNode items that are
    in the subtreee of the notified PolicyNode and queues reconciliation for
    those PolicyNode objects.

### Life of an update to controlled state (eg Role, RoleBinding)

1.  Any item in a module controlled resource is mutated
1.  The kubebuilder framework recieves the mutation event and hands it to our
    custom funciton which we registered via WatchTransformationOf.
1.  We translate the event's namespace to the name of the PolicyNode which
    controls the namespace and kubebuilder queues a reconcile event for the
    PolicyNode

### Life of an update to a Namespace

1.  A namespace is mutated
1.  The kubebuilder framework triggers a reconcile for the PolicyNode with the
    namepspace name.

### Life of a Reconciliation Event (PolicyNode)

1.  Kubebuilder takes an item off the queue
1.  Kubebuilder calls the reconcile function with the item which refers to the
    name of a PolicyNode.
1.  The reconcile function checks if the namespace is reserved and if so returns
    an error.
1.  Ancestry is fecthed for the PolicyNode, if the ancestry is not complete this
    results in returning an error to kubebuilder. If the node does not exist, we
    explicitly delete the namespace which corresponds to the PolicyNode. If the
    node itself is a Abstract Namespace, no reconciliation occurs.
1.  The namespace for the PolicyNode is created if it does not yet exist.
1.  Each module generates the computed policy from the PolicyNode Ancestry for
    the controlled resource in the Namespace corresponding to the PolicyNode.
1.  The computed policy is compared to the policy on the API server and the
    appropriate API operations are made to bring the API server's policies to
    the intended state.``
