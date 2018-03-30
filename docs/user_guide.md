# Nomos User Guide

## Glossary



*   **Workload cluster:** A cluster where a user runs a pod.
*   **Enrolled cluster:** A workload cluster running Nomos components.


## Overview

In environments with many users spread across many teams, having multiple tenants within a cluster, allocated into namespaces, maximizes resource utilization while providing isolation. However, as the number of namespaces grows, it becomes increasingly hard for cluster operators to manage per-namespace policies like authorization (RBAC Roles/Rolebinding) and quota (ResourceQuota), etc. In addition, real world deployments often require multiple clusters in order to tolerate region failures, reduce network latencies for end users, or simply to scale beyond the current size limits of a Kubernetes cluster. Nomos makes it easier to manage large multi-tenant and multi-cluster deployments, by reducing the load on cluster operators and reducing the surface area to secure.

At a high level, Nomos serves two separate functions:



*   **Policy distribution**: Distribute policy definitions from a centralized source of truth to all workload clusters. The extensible policy distribution mechanism allows Nomos to support a wide range of technologies implementing the source of truth for policies (e.g. YAML files in Git, Google Cloud IAM & admin, Active Directory, etc.)
*   **Hierarchical policy enforcement**: Group together namespaces and associated policies into a hierarchy modelled after how departments and teams are organized. Nomos provides a set of controllers running on the workload clusters that consume hierarchical policies definitions and are responsible for enforcing them.

In this release, there are four main areas that Nomos helps manage:



1.  **Namespaces**:  With Nomos you have one set of namespaces that apply to all clusters.   Nomos also introduces hierarchical namespace support giving you the ability to group namespaces together for common policies and to facilitate delegation.  In Nomos, only leaf namespaces can contain non-policy resources, while the intermediate and root nodes provide policy attach points for policies such as RBAC, ResourceQuota, and more.
1.  **Hierarchical RBAC policies**:  Nomos provides central management of RBAC policies and enables inheritance of namespace-level RBAC resources. For example a Rolebinding from an ancestor is inherited by all descendent namespaces, removing duplication.
1.  **Hierarchical ResourceQuota policies**:  With Nomos one can manage quota centrally and set quota hierarchically.
1.  **Cluster-level policies:** In addition to namespace-level policies, Nomos allows you to centrally manage cluster-level policies such as ClusterRole/Rolebinding and PodSecurityPolicy.

This guide will take you through managing each of these resources.


### Nomos Concepts


#### Namespaces

In Kubernetes, namespaces are the isolation construct for implementing multi-tenancy and are the parents of all workload resources (pods, replica sets, service accounts, etc). Namespaces can be used to isolate workloads that have the same set of human owners as well as to isolate different workload environments.

Generally anytime you want to have a workload managed by a distinct person or set of people (e.g. on-call person only for prod workloads, whole development team for dev workloads), it makes sense to create a new namespace.  If you want to have a common person or set of people be able to perform the same set of operations within a set of namespaces, create a policyspace.

Nomos with its hierarchical control allows many namespaces to be managed by the same set of people so it's possible to create more granular namespaces for a team of people without incurring additional policy administration overhead.


#### Policyspaces

With Nomos, we give admins the ability to group namespaces together and to form groups of groups through a hierarchy.  We call a non-leaf node in this tree, whose leaves are namespaces, a  policyspace. We can think of policyspaces as Organization Units (or Policy Information Point in [XACML](https://en.wikipedia.org/wiki/XACML#Terminology) parlance).  They exist to delegate policy control to sub organization leaders. This approach has been long established by LDAP and Active Directory.

Policyspaces can be parents of policyspaces and namespaces. Policyspace and namespaces must both be globally unique.


#### Delegation

As the organization complexity grows, it is important for an admin to have the ability to delegate administration of a subset of the hierarchy to another admin. The mechanism for delegation is specific to the source of truth. For example, on Google Cloud Platform, an admin can grant setIamPolicy permission to someone who can then set policies independently. In Git, the admin can give commit permissions to a subtree to someone else.


#### Example

One can imagine a brick and mortar retailer having a hierarchy of policyspaces and namespaces that looks like this

![drawing](img/foo_corp_hierarchy.svg)

Figure 1: foo-corp organization

In Foo corp, a small team (8-10 people) runs a few microservice workloads that together provide a bigger component.  Everyone on the team has the same set of roles (e.g. deployment, on-call, coding, etc).  We also assume that the small team will run a dev and staging setup for qualification and will want to ensure that these environments have different security postures. Also these workloads need to run in multiple regions.

This gives the ability for the Shipping App Backend team to manage three different namespaces but only have to maintain one authorization policy for team members.  Each of their namespaces is isolated by environment, allowing identically-named objects in the three envionments' instantiations of the backend stack, as well as providing tighter security, e.g. allowing one namespace to have additional authorized users but not the others, and allocating private quota to each namespace. 


### System Overview

![drawing](img/nomos_arch.svg) 


The above diagram is a simplified view of Nomos components running on a workload cluster. Each component is described below.


#### PolicyImporter

PolicyImporter is an abstraction for a controller that consumes policy definitions from an external source of truth and builds a canonical representation of the hierarchy using cluster-level CRD(s) defined by Nomos. Nomos can be extended to support different sources of truth (e.g. Git, GCP, Active Directory) using different implementations of this abstraction. Note that we treat this canonical representation as internal implementation which should not be directly consumed by users.


#### Syncer

A set of controllers (currently packaged as a single binary) that consume the canonical representation of the hierarchy produced by PolicyImporter and perform CRUD on namespaces and native K8S policy resources such as Role/Rolebinding, ResourceQuota, etc.


#### ResourceQuotaAdmissionController

A ValidatingAdmissionWebhook that enforces hierarchical quota policies which providers hierarchical quota on top of existing ResourceQuota admission controller.


## Set Up

There's a one-time set up required to set up Nomos components described in this section. The user running these commands should have a cluster-admin rolebinding.


### Cluster Requirements

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

**Warning:** In the current release of Nomos, we require that all namespaces be managed by Nomos. It is recommended to create a new cluster for use with Nomos.


### Installing Nomos

Download the Nomos installer script to a directory on your machine.


```
cd
mkdir -p tmp/nomos
cd tmp/nomos
curl https://storage.googleapis.com/nomos-release/run-installer.sh -o run-installer.sh
chmod +x run-installer.sh
```


The installer can run in two modes: batch (the default) and interactive (menu-driven).


#### Batch installation

Batch installation is based on reading a prepared configuration file. A sample configuration is provided below. Most fields have been preset to point at the sample configuration repository, but some you must supply, as noted below. 

Note:



*   You must edit the contexts field to point at the list of clusters you want to install on.  Use kubectl config get-contexts to see what contexts are available to you.
*   You must edit the user field to be set to your username that is valid for authenticating to the clusters.  This username must be valid on all clusters included in the contexts field.
*   Please do NOT replace $HOME with the name of your actual home directory, leave it as a placeholder.
*   Update GIT_SYNC_REPO to point to your repo if you're not using the sample repo provided

```
contexts:
  - kubeconfig_context_of_your_cluster
git:
  GIT_SYNC_BRANCH: master
  GIT_SYNC_REPO: git@github.com:foo-corp/nomos-policies.git
  GIT_SYNC_WAIT: 60
  ROOT_POLICY_DIR: foo-corp
ssh:
  knownHostsFilename: $HOME/.ssh/known_hosts
  privateKeyFilename: $HOME/.ssh/id_rsa.nomos
user: youruser@foo-corp.com
```



Once you have finished with editing the file, you can run the batch installer as follows:


```
./run-installer.sh --config=/path/to/your/config.yaml
```


#### Interactive installation

Interactive installation is menu driven.  It allows you to edit the configuration through a user-friendly menu as shown in the figure below:

![drawing](img/installer_interactive.png)

You can start from a sample configuration:


```
./run-installer.sh --interactive
```


or you can start from an existing configuration, similar to the one in the [Batch Installation](#batch-installation) section.


```
./run-installer.sh --interactive --config=/path/to/your/config.yaml
```


"Save" will store the current settings that you can then reuse later in a batch installation. "Install" will run the installer on the chosen clusters.


#### Verify installation 

To verify that Nomos components are correctly installed, issue the following command and verify that all components listed have status displayed as "Running."

Check running components:


```
$ kubectl get pods -n=nomos-system
NAME                                                  READY     STATUS    RESTARTS   AGE
git-policy-importer-66bf6b9db4-pbsxn                  2/2       Running   0          24m
resourcequota-admission-controller-64988d97f4-nxmsc   1/1       Running   0          24m
syncer-58545bc77d-l485n                               1/1       Running   0          24m
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
nomos-system       Active    2m

```



### Uninstalling

Uninstall will shut down Nomos components, but will leave the cluster state otherwise intact.  


```
./run-installer.sh --config=/path/to/your/config.yaml --uninstall=true
```


You will also need to supply a boolean flag `--yes` to confirm uninstallation.


## Using Nomos


### Policy Hierarchy Operations


#### Creation

When using Git as source of truth, we represent the hierarchy of policyspaces and namespaces using the filesystem hierarchy. 

Following the [example](#example) above, we can have such a directory structure ([Available on this GitHub repo](https://github.com/frankfarzan/foo-corp-example)):

```
foo-corp
|-- audit
|   `-- namespace.yaml
|-- online
|   `-- shipping-app-backend
|       |-- shipping-dev
|       |   |-- job-creator-rolebinding.yaml
|       |   |-- job-creator-role.yaml
|       |   |-- namespace.yaml
|       |   `-- quota.yaml
|       |-- shipping-prod
|       |   `-- namespace.yaml
|       |-- shipping-staging
|       |   `-- namespace.yaml
|       |-- pod-creator-rolebinding.yaml
|       `-- quota.yaml
|-- namespace-reader-clusterrolebinding.yaml
|-- namespace-reader-clusterrole.yaml
|-- pod-creator-clusterrole.yaml
|-- pod-security-policy.yaml
`-- viewers-rolebinding.yaml
```


Figure 2: The resulting directory structure.


###### Definitions



1.  A leaf directory represents a namespace.
1.  A non-leaf directory represents a policyspace.


###### Constraints



1.  A namespace directory must contain a Namespace resource.
1.  A namespace directory can contain any number of Role and Rolebinding resources, and a single ResourceQuota resource.
1.  A namespace directory name must match the namespace name in all resources in that directory.
1.  A policyspace directory must not contain a Namespace resource.
1.  A policyspace directory can contain any number of Rolebinding resources and a single ResourceQuota resource but must not contain Roles. These resources must not specify a namespace.
1.  The root policyspace directory can also contain any number of ClusterRole, ClusterRolebinding, and PodSecurityPolicy resources.
1.  Both policyspace and namespace directory names must be valid Kubernetes namespace names (i.e. [DNS Label](https://github.com/kubernetes/community/blob/master/contributors/design-proposals/architecture/identifiers.md)) and must be unique in the hierarchy. In addition a name cannot be "default", "nomos-system", or have "kube-" prefix.
1.  Any other file not explicitly mentioned above is ignored by Nomos in this release (e.g. OWNERS files).

There are no requirements on file names or how many resources are packed in a file.

Nomos provides a validation tool which should be used as presubmit check (e.g. Using Git's [pre-submit hook](https://git-scm.com/book/en/v2/Customizing-Git-Git-Hooks) or Github's [required status check](https://help.github.com/articles/about-required-status-checks/)) before committing any policy changes:


```
$ nomosvet /path/to/foo-corp
```


When a valid tree is committed to Git and synced, Nomos controllers automatically create namespaces and corresponding policy resources to enforce hierarchical policy. In this example, Nomos automatically creates "shipping-dev", "shipping-staging", and "shipping-prod" namespaces. We discuss specific policy types and their enforcement in later sections.

Note that when using Git as source of truth, it is up to the repo owners to set proper access control mechanism (e.g. using OWNERS or CODEOWNER files) to ensure right people can approve/review/commit policy changes. It is recommended to use a hierarchical access control mechanism such as OWNERS file in order to delegate policy changes instead of requiring a central authority to approve all changes.


#### Deletion

Deleting a namespace directory is a very destructive operation.  All resources including identities, policies and workload resources will be deleted on every cluster where this namespace is present. Similarly deleting a policyspace directory recursively, deletes all descendаnt names and associated resources.


#### Renaming

Renaming a namespace directory (which requires renaming Namespace name as well) is destructive since it deletes that namespace and creates a new namespace.

Renaming a policyspace directory has no externally visible effect.


#### Moving

Moving a policyspace or namespace directory can lead to policy changes in namespaces, but does not delete a namespace or workload resources.


### Policy Types


#### Namespace-level Policies


###### Role/Rolebinding

Nomos enables RBAC policies to be applied hierarchically following these properties:



1.  A RoleBinding specified in a policyspace is inherited by all descendant namespaces
1.  A Role cannot be specified in a policyspace. If multiple namespaces need to refer to the same role, use a ClusterRole.
1.  A RoleBinding can be specified in a namespace (Existing K8S behavior)
1.  A Role can be specified in a namespace (Existing K8S behavior). 

For example, we can create a RoleBinding in "shipping-app-backend" policyspace such that anyone belonging to "shipping-app-backend-team" group is able to create pods in all namespace descendants (i.e. "shipping-dev", "shipping-staging", "shipping-prod"):

 


```
$ cat foo-corp/online/shipping-app-backend/pod-creator-rolebinding.yaml 
kind: RoleBinding
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: pod-creators
subjects:
- kind: Group
  name: shipping-app-backend-team
  apiGroup: rbac.authorization.k8s.io
roleRef:
  kind: ClusterRole
  name: pod-creator
  apiGroup: rbac.authorization.k8s.io                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                          
```


This is done by automatically creating inherited RoleBindings in a namespace:


```
$ kubectl get rolebinding --namespace shipping-dev -o name
job-creators
pod-creators
```


Inheritance is implemented by flattening resources in namespaces. In "shipping-dev" namespace, "pod-creators" is inherited and "job-creators" is created directly in the namespace.

Note that policies are themselves resources which means a user may be able to edit policies outside of Nomos (e.g. using kubectl) or create rolebindings subject to [privilege escalation prevention](https://kubernetes.io/docs/admin/authorization/rbac/#privilege-escalation-prevention-and-bootstrapping) in Kubernetes.


###### ResourceQuota

A quota set on a namespace behaves just like it does in native kubernetes, restricting the specified resources. In Nomos you can also set resource quota on policyspaces. This will set the quota limit on all the namespaces that are children of the provided policyspace within a single cluster. The policyspace limit ensures that the sum of all the resources of a specified type in all the children of the policyspace do not exceed the specified quota. Quota is evaluated in a hierarchical fashion starting from the namespace, up the policyspace hierarchy - this means that a quota violation at any level will result in a Forbidden exception.

A quota is allowed to be set to immediately be in violation. For example, when a workload namespace has 11 pods, we can still set quota to "pods: 10" in a parent policyspace, creating an overage. If a workload namespace is in violation, the ResourceQuotaAdmissionController will prevent new objects of that type from being created until the total object count falls below the quota limit, but existing objects will still be valid and operational.

Here we add hard quota limit on number of pods across all namespaces having "shipping-app-backend" as an ancestor:


```
$ cat foo-corp/online/shipping-app-backend/quota.yaml
kind: ResourceQuota
apiVersion: v1
metadata:
  name: pod-quota
spec:
  hard:
    pods: "3"
```


In this case, total number of pods allowed in "shipping-prod", "shipping-dev", and "shipping-staging" is 3. When creating the fourth pod (e.g. in "shipping-prod"), you will see the following error:


```
Error from server (Forbidden): exceeded quota in policyspace "shipping-app-backend", requested: pods=4, limit: pods=3
```



#### Cluster-level Policies

Cluster-level policies will function in the same manner as in a vanilla kubernetes cluster with the only addition being that Nomos will distribute and manage them on the workload clusters.

Cluster-level policies must be placed immediately within the root policyspace directory. Since cluster-level policies have far-reaching effect, they should only be editable by cluster admins.


###### ClusterRole/ClusterRoleBinding

For example, we can create namespace-reader ClusterRole: 


```
$ cat foo-corp/namespace-viewer-role.yaml
kind: ClusterRole
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: namespace-reader
rules:
- apiGroups: [""]
  resources: ["namespaces"]
  verbs: ["get", "watch", "list"]
```


And a ClusterRoleBinding referencing this Role:


```
$ cat foo-corp/namespace-viewer-rolebinding.yaml
kind: ClusterRoleBinding
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: namespace-readers
subjects:
- kind: User
  name: cheryl@foo-corp.com
  apiGroup: rbac.authorization.k8s.io
roleRef:
  kind: ClusterRole
  name: namespace-reader
  apiGroup: rbac.authorization.k8s.io
```



###### PodSecurityPolicy 

PodSecurityPolicies are created in the same manner as other cluster level resources:


```
cat foo-corp/pod-security-policy.yaml 
apiVersion: extensions/v1beta1
kind: PodSecurityPolicy
metadata:
  name: psp
spec:
  privileged: false
  seLinux:
    rule: RunAsAny
  supplementalGroups:
    rule: RunAsAny
  runAsUser:
    rule: RunAsAny
  fsGroup:
    rule: RunAsAny
  volumes:
  - '*'
```



### Monitoring and Debugging


#### Logging

Nomos follows [K8S logging convention](https://github.com/kubernetes/community/blob/master/contributors/devel/logging.md). By default, all binaries log at V(2).

List all nomos-system pods:


```
$ kubectl get deployment -n nomos-system
NAME                                 DESIRED   CURRENT   UP-TO-DATE   AVAILABLE   AGE
git-policy-importer                  1         1         1            1           13d
resourcequota-admission-controller   1         1         1            1           9d
syncer                               1         1         1            1           13d
```


To see logs for pod:


```
$ kubectl logs syncer -n nomos-system
```



### Future Features


#### Cluster Targeting

Currently, Nomos distributes identical policies to every cluster. We want to enable specifying cluster-specific policies. For example, a namespace should be able to have a different quota in clusters A and B. We can also not sync a namespace to a cluster at all.


#### Enforcement Modes

Currently, it's possible to have un-managed resources that are not declared in the source of truth. For example, a cluster-admin can create a namespace outside of Nomos. In the future, we want to create various modes for reporting or disallowing such resources.


### Appendix


#### Nomos CRDs 


```
// These comments must remain outside the package docstring.
// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// PolicyNode is the top-level object for the policy node data definition.
//
// It holds a policy defined for a single org unit (namespace).
type PolicyNode struct {
	metav1.TypeMeta `json:",inline"`
	// Standard object's metadata. The Name field of the policy node must
// match the namespace name.
	// +optional
	metav1.ObjectMeta `json:"metadata"`
	// The actual object definition, per K8S object definition style.
	Spec PolicyNodeSpec `json:"spec"`
}

// PolicyNodeSpec contains all the information about a policy linkage.
type PolicyNodeSpec struct {
	// False for leaf namespaces where pods will actually be scheduled,
	// True for the parent org unit namespace where this policy is linked
	// to, but no containers should run
	Policyspace bool `json:"policyspace"`
	// The parent org unit
	Parent string `json:"parent"`
	// The policies attached to that node
	Policies Policies `json:"policies"`
}

// Policies contains all the defined policies that are linked to a particular
// PolicyNode.
type Policies struct {
	Roles         []rbac_v1.Role            `json:"roles"`
	RoleBindings  []rbac_v1.RoleBinding     `json:"roleBindings"`
	ResourceQuota core_v1.ResourceQuotaSpec `json:"resourceQuota"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// PolicyNodeList holds a list of namespace policies, as response to a List
// call on the policy hierarchy API.
type PolicyNodeList struct {
	metav1.TypeMeta `json:",inline"`
	// Standard object's metadata.
	// +optional
	metav1.ListMeta `json:"metadata,omitempty"`
	// Items is a list of policy nodes that apply.
	Items []PolicyNode `json:"items"`
}
// ClusterPolicy is the top-level object for the policy node data definition.
//
// It holds a policy defined for a single org unit (namespace).
type ClusterPolicy struct {
	metav1.TypeMeta `json:",inline"`
	// Standard object's metadata. The Name field of the policy node must
// match the namespace name.
	// +optional
	metav1.ObjectMeta `json:"metadata"`
	// The actual object definition, per K8S object definition style.
	Spec ClusterPolicySpec `json:"spec"`
}

// ClusterPolicySpec defines the policies that will exist at the cluster level.
type ClusterPolicySpec struct {
	// Sources describes the resource / name / resourceVersion of
// definitions that were merged to
	// create this object, for example ["clusterpolicy.prod.275564"].
// Note that there is no ambiguity
	// in this as the resource name and resource version are not allowed
// to contain the '.' character.
	// This field will not be set in the MasterPolicyNode and will only be
// set at enrolled clusters.
	Sources []string `json:"sources"`
	// The policies specified for cluster level resources.
	Policies ClusterPolicies `json:"policies"`
}

// ClusterPolicies specifies the policies nomos synchronizes to a cluster. This is factored out
// due to the fact that it is specified in MasterClusterPolicyNodeSpec and ClusterPolicyNodeSpec.
type ClusterPolicies struct {
	// Type defines the type of resources that this holds.
// It will hold one of the cluster scoped
	// resources and should have a resource name that matches the resource type it holds.
	Type string `json:"type"`
	// Cluster scope resources.
	ClusterRoles        []rbac_v1.ClusterRole                  `json:"clusterRoles"`
	ClusterRoleBindings []rbac_v1.ClusterRoleBinding           `json:"clusterRoleBindings"`
	PodSecurtiyPolicies []extensions_v1beta1.PodSecurityPolicy `json:"podSecurityPolicy"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// ClusterPolicyList holds a list of cluster level policies, returned as response to a List
// call on the cluster policy hierarchy.
type ClusterPolicyList struct {
	metav1.TypeMeta `json:",inline"`
	// Standard object's metadata.
	// +optional
	metav1.ListMeta `json:"metadata,omitempty"`
	// Items is a list of policy nodes that apply.
	Items []ClusterPolicy `json:"items"`
}

