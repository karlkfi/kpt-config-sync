package adapter

import (
	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/apis/rbac"
)

// A policy node which maps to a kubernetes namespace to which policy is attached
type PolicyNode struct {
	Name             string      `json:name'`            // The name of the org unit or the namespace
	WorkingNamespace bool        `json:workingNamespace` // True for leaf namespaces where pods will actually be scheduled, false for the parent org unit namespace where policy is attached but no containers should run
	Parent           string      `json:parent`           // The parent org unit
	Policies         PolicyLists `json:policies`         // The policies attached to that node

}

type PolicyLists struct {
	Roles          []rbac.Role             `json:roles`
	RoleBindings   []rbac.RoleBinding      `json:roleBindings`
	ResourceQuotas []api.ResourceQuotaSpec `json:resourceQuotas`
}
