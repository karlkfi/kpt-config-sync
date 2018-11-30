package initialize

const (
	repoFile     = "repo.yaml"
	repoContents = `kind: Repo
apiVersion: nomos.dev/v1alpha1
metadata:
  name: repo
spec:
  version: "0.1.0"
`
	rbacSyncFile     = "rbac-sync.yaml"
	rbacSyncContents = `kind: Sync
apiVersion: nomos.dev/v1alpha1
metadata:
  name: rbac
spec:
  groups:
  - group: rbac.authorization.k8s.io
    kinds:
    - kind: ClusterRole
      versions:
      - version: v1
        compareFields:
        - rules
    - kind: ClusterRoleBinding
      versions:
      - version: v1
        compareFields:
        - subjects
        - roleRef
    - kind: Role
      versions:
      - version: v1
        compareFields:
        - rules
    - kind: RoleBinding
      versions:
      - version: v1
        compareFields:
        - subjects
        - roleRef
`
	resourceQuotaSyncFile     = "resourcequota-sync.yaml"
	resourceQuotaSyncContents = `kind: Sync
apiVersion: nomos.dev/v1alpha1
metadata:
  name: resourceQuota
spec:
  groups:
  - kinds:
    - kind: ResourceQuota
      versions:
      - version: v1
`
	podSecuritySyncFile     = "podsecuritypolicy-sync.yaml"
	podSecuritySyncContents = `kind: Sync
apiVersion: nomos.dev/v1alpha1
metadata:
  name: podsecuritypolicies
spec:
  groups:
  - group: extensions
    kinds:
    - kind: PodSecurityPolicy
      versions:
      - version: v1beta1
`
	readmeFile         = "README.md"
	rootReadmeContents = `# GKE Policy Management Directory

This is the root directory for GKE Policy Management for your cluster.

* See [system/](system/README.md) for system configuration.
* See [cluster/](cluster/README.md) for cluster-scoped resources.
* See [namespaces/](namespaces/README.md) for namespace-scoped resources.
`
	systemReadmeContents = `# System

This directory contains system configs such as the repo version and how resources are synced.
`
	clusterReadmeContents = `# Cluster

This directory contains cluster-scoped resources.
`
	namespacesReadmeContents = `# Namespaces

This directory contains namespace-scoped resources.
`
)
