package initialize

const (
	repoFile     = "repo.yaml"
	repoContents = `kind: Repo
apiVersion: nomos.dev/v1
metadata:
  name: repo
spec:
  version: "0.1.0"
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
