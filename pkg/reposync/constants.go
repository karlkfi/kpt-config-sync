package reposync

import "sigs.k8s.io/controller-runtime/pkg/client"

// Name is the required name of any RepoSync CR.
const Name = "repo-sync"

// ObjectKey returns a key appropriate for fetching a RepoSync in the given
// namespace.
func ObjectKey(namespace string) client.ObjectKey {
	return client.ObjectKey{
		Namespace: namespace,
		Name:      Name,
	}
}
