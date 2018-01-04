/*
Copyright 2017 The Stolos Authors.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

		http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package authorizer implements a hierarchical authorization module for
// Stolos.
//
// Authorizer is the top-level object.  The hierarchical authorization
// is deployed as a webhook which is why this authorizer takes API objects
// (SubjectAccessReview*) as request and produces the SubjectAccessReviewStatus
// as result.  The hierarchical authorization uses under the hood the regular
// RBAC authorizer, but adapts to it a data source that draws the roles and
// bindings from the Stolos hierarchical policy structure.
//
// The data source is implemented using the Informer mechanism from Kubernetes.
// The informers pull the PolicyNode information directly from the respective
// CRD stored in the API server in the local cluster.
package authorizer

import (
	"fmt"
	"reflect"

	"github.com/davecgh/go-spew/spew"
	"github.com/golang/glog"
	policyhierarchy "github.com/google/stolos/pkg/api/policyhierarchy/v1"
	rules "github.com/google/stolos/pkg/client/rules"
	"github.com/google/stolos/pkg/util/set/stringset"
	"github.com/pkg/errors"
	authz "k8s.io/api/authorization/v1beta1"
	apicore "k8s.io/api/core/v1"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apiserver/pkg/authentication/user"
	"k8s.io/apiserver/pkg/authorization/authorizer"
	"k8s.io/client-go/tools/cache"
	apisrbac "k8s.io/kubernetes/pkg/apis/rbac"
	apisrbacconv "k8s.io/kubernetes/pkg/apis/rbac/v1"
	"k8s.io/kubernetes/pkg/registry/rbac/validation"
	"k8s.io/kubernetes/plugin/pkg/auth/authorizer/rbac"
)

var (
	// TypeMeta is the SubjectAccessReview type meta.
	TypeMeta = meta.TypeMeta{
		Kind:       "SubjectAccessReview",
		APIVersion: "authorization.k8s.io/v1beta1",
	}
)

// rbacInformerAdapter adapts a SharedIndexInformer for the hierarchical
// policies to the {,Cluster}Role{Getter,BindingLister}
type rbacInformerAdapter struct {
	informer cache.SharedIndexInformer
}

var _ validation.ClusterRoleGetter = (*rbacInformerAdapter)(nil)

// GetClusterRoleName always returns nothing found.  The hierarchical
// authorizer does not deal with cluster roles.  Implements validation.ClusterRoleGetter.
func (r *rbacInformerAdapter) GetClusterRole(name string) (*apisrbac.ClusterRole, error) {
	return &apisrbac.ClusterRole{}, nil
}

var _ validation.ClusterRoleBindingLister = (*rbacInformerAdapter)(nil)

// ListClusterRoleBindings always returns nothing found.  The hierarchical
// authorizer does not deal with cluster role bindings. Implements
// validation.ClusterRoleBindingLister.
func (r *rbacInformerAdapter) ListClusterRoleBindings() ([]*apisrbac.ClusterRoleBinding, error) {
	return []*apisrbac.ClusterRoleBinding{}, nil
}

var _ validation.RoleGetter = (*rbacInformerAdapter)(nil)

// GetRole implements validation.RoleGetter.
func (r *rbacInformerAdapter) GetRole(namespace, name string) (*apisrbac.Role, error) {
	policies, err := r.policyRulesFor(namespace)
	if err != nil {
		return nil, errors.Wrapf(
			err, "while looking up role: namespace=%v, name=%v", namespace, name)
	}

	// A role closer to the root namespace is the more important one.
	// NOTE(fmil): PERFORMANCE Yes indeed, this is a slow way to get the
	// policies, but it is very simple.  We may want, at some point, to index
	// the policies for quick access.
	var role *apisrbac.Role
	for _, policy := range *policies {
		for _, candidate := range policy.Policies.Roles {
			if candidate.Name == name {
				var newRole apisrbac.Role
				apisrbacconv.Convert_v1_Role_To_rbac_Role(
					&candidate,
					&newRole,
					nil, /*scope; what is it good for?!*/
				)
				expandResourceQuotaPermissions(&newRole)
				role = &newRole
			}
		}
	}
	if role == nil {
		// Not found.  Should we return nil?  Or a default?  Unknown.
		return nil,
			fmt.Errorf("role not found: namespace=%v, name=%v", namespace, name)
	}
	glog.V(4).Infof("GetRole found role: %v", spew.Sdump(*role))
	return role, nil
}

// matchesCoreResourceQuota returns true if the cross-product of api groups and
// resource types specified includes the resource quota resource from the core
// API group.
func matchesCoreResourceQuota(apiGroups []string, resources []string) bool {
	apiGroupSet := stringset.NewFromSlice(apiGroups)
	glog.V(1).Infof("false: %v", apiGroupSet)
	if !apiGroupSet.Contains("") && !apiGroupSet.Contains("*") {
		// If the api groups don't refer to the core API group, then no quotas
		// are involved.
		glog.V(1).Infof("false")
		return false
	}
	resourceSet := stringset.NewFromSlice(resources)
	result := resourceSet.Contains(string(apicore.ResourceQuotas)) &&
		!resourceSet.Contains(string(policyhierarchy.StolosResourceQuotaResource))
	glog.V(1).Infof("result: %v", result)
	return result
}

// newStolosPolicyRule creates a new policy rule that applies to Stolos
// resource quotas, based on a policy rule for "regular" policy rules.
// Requires 'prototype' to be a policy rule that applies to regular resource
// quota.
func newStolosPolicyRule(prototype *apisrbac.PolicyRule) *apisrbac.PolicyRule {
	result := prototype.DeepCopy()
	result.APIGroups = []string{policyhierarchy.GroupName}
	result.Resources = []string{policyhierarchy.StolosResourceQuotaResource}
	return result
}

// expandResourceQuotaPermissions grants to objects of the type
// "k8us.k8s.io"/stolosresourcequota" the same permissions that are granted for
// ""/resourcequota.
//
// This is done to ensure that the same set of policy rules apply to all
// resource quota objects, regardless of whether they are native Kubernetes
// resource quotas, or are hierarchical quotas added by Stolos.  This ensures
// that any user that is permitted to access Kubernetes resource quota can
// also access the hierarchical resource quota.
//
// If, however, one wants to add additional permission rules on the
// hierarchical resource quota, such rules can be added just as for any other
// custom resource.
func expandResourceQuotaPermissions(rb *apisrbac.Role) {
	glog.V(6).Infof("Role at entry: %+v", *rb)
	for i, rule := range rb.Rules {
		if len(rule.ResourceNames) != 0 {
			// If the resources are explicitly named, don't modify the role to
			// include stolos resource quotas.
			glog.V(8).Infof("Skipping rule: i=%v, rule=%#v", i, rule)
			continue
		}
		resources := stringset.NewFromSlice(rule.Resources)
		glog.V(7).Infof("resources=%v", resources)
		if !matchesCoreResourceQuota(rule.APIGroups, rule.Resources) {
			// If we already inserted the stolos resource quotas, or if no
			// resource quotas are mentioned at all, skip this rule.
			continue
		}
		rb.Rules = append(rb.Rules, *newStolosPolicyRule(&rule))
	}
	glog.V(6).Infof("Resulting role: %+v", *rb)
}

var _ validation.RoleBindingLister = (*rbacInformerAdapter)(nil)

// ListRoleBindings implments validation.RoleBindingsLister.
func (r *rbacInformerAdapter) ListRoleBindings(namespace string) ([]*apisrbac.RoleBinding, error) {
	policies, err := r.policyRulesFor(namespace)
	if err != nil {
		return nil, errors.Wrapf(
			err, "while looking up role bindings: namespace=%v", namespace)
	}

	var result []*apisrbac.RoleBinding
	// Create with capacity that expects up to 5 role bindings per hierarchy
	// level.  See performance notes in GetRole above.
	result = make([]*apisrbac.RoleBinding, 0, 5*len(*policies))
	for _, policy := range *policies {
		for _, candidate := range policy.Policies.RoleBindings {
			var newRoleBinding apisrbac.RoleBinding
			apisrbacconv.Convert_v1_RoleBinding_To_rbac_RoleBinding(
				&candidate,
				&newRoleBinding,
				nil, /* scope */
			)
			result = append(result, &newRoleBinding)
		}
	}
	glog.V(6).Infof("ListRoleBindings found role bindings:\n\t%v",
		spew.Sdump(result))
	return result, nil
}

// policyRulesFor lists all policy rules that apply for the given 'namespace'.
// The returned PolicyNodeSpec index 0 is the policy node spec for the leaf
// namespace.  The last is for the root namespace.
func (r *rbacInformerAdapter) policyRulesFor(
	namespace string) (*[]policyhierarchy.PolicyNodeSpec, error) {
	return rules.GetPolicyRules(r.informer, namespace)
}

// Authorizer deals with the authorization mechanics.  Instantiate
// using a New() call below.
type Authorizer struct {
	// The actual authorizer that does the heavy-lifting authorization.
	delegateAuthz *rbac.RBACAuthorizer

	// Looks up the roles and role bindings for the Authorizer.
	adapter rbacInformerAdapter
}

// New creates an authorizer that watches the supplied informer for changes in
// the policy nodes structure.
func New(informer cache.SharedIndexInformer) *Authorizer {
	adapter := rbacInformerAdapter{informer}
	return &Authorizer{
		adapter:       adapter,
		delegateAuthz: rbac.New(&adapter, &adapter, &adapter, &adapter),
	}
}

// Authorize verifies whether 'request' is allowed based on the current security
// context and the spec to be reviewed.  Returns SubjectAccessReviewStatus with
// the verdict.
func (a *Authorizer) Authorize(
	request *authz.SubjectAccessReviewSpec) *authz.SubjectAccessReviewStatus {
	if request == nil {
		panic("request is nil")
	}
	// Adapt the incoming request to authorization.Attributes interface.  This
	// is the inverse conversion to that already performed by the webhook
	// authorizer.
	attributes := NewAttributes(request)
	verdict, reason, err := a.delegateAuthz.Authorize(attributes)
	status := toSubjectAccessReviewStatus(verdict, reason, err)
	glog.V(4).Infof("Authorize: request=%+v, status=%+v",
		spew.Sdump(request), spew.Sdump(status))
	return status
}

// toSubjectAccessReviewStatus converts the delegated authorizer results back to
// something that can be communicated back to the apiserver.
func toSubjectAccessReviewStatus(verdict authorizer.Decision, reason string, err error) *authz.SubjectAccessReviewStatus {
	var evaluationError string
	if err != nil {
		verdict = authorizer.DecisionDeny
		reason = "evaluation error"
		evaluationError = fmt.Sprintf("webhook authz error: %v", err)
	}
	result := authz.SubjectAccessReviewStatus{
		Allowed:         verdict == authorizer.DecisionAllow,
		Reason:          reason,
		EvaluationError: evaluationError,
	}
	return &result
}

// ----------------------------------------------------------------------
// This section implements authorizer.Attributes.
var _ authorizer.Attributes = (*Attributes)(nil)

// Attributes is an object that adapts the SubjectAccessReviewSpec to an
// authorizer.Attributes.
//
// This conversion is useful because we can then reuse the entire "regular"
// RBAC authorizer by simply feeding it the hierarchically evaluated Role and
// RoleBindings.
type Attributes struct {
	request *authz.SubjectAccessReviewSpec
}

// NewAttributes wraps request into an authorizer.Attributes object.  Use to
// adapt a webhook authorizer request to a non-webhook authorizer request.
func NewAttributes(request *authz.SubjectAccessReviewSpec) *Attributes {
	return &Attributes{request}
}

// GetUser implements authorizer.Attributes.
func (a *Attributes) GetUser() user.Info {
	req := a.request
	result := &user.DefaultInfo{
		Name:   req.User,
		UID:    req.UID,
		Groups: req.Groups,
		Extra:  convertFromSarExtra(req.Extra),
	}
	glog.V(9).Infof("GetUser() -> %+v", result)
	return result
}

func (a *Attributes) GetVerb() string {
	// This requires IsResourceRequest == true
	return a.request.ResourceAttributes.Verb
}

var (
	// emptyResourceRequest is used to detect a request that does not refer to
	// a resource.
	emptyResourceRequest authz.ResourceAttributes = authz.ResourceAttributes{}

	// emptyNonResourceAttributes is same as above, but for non-resources.
	emptyNonResourceAttributes = authz.NonResourceAttributes{}
)

// IsResourceRequest implements authorizer.Attributes.
func (a *Attributes) IsResourceRequest() bool {
	// TODO(fmil): Maybe don't use reflect for this for performance concerns?
	// The logic below was inferred from
	// k8s.io/apiserver/plugin/pkg/authorizer/webhook/webhook.go.  There does
	// not seem to be a spec that enforces the exclusivity of the two items
	// below, but it's implied in the webhook code.
	return !reflect.DeepEqual(
		a.request.ResourceAttributes, emptyResourceRequest) &&
		(reflect.DeepEqual(
			a.request.NonResourceAttributes, emptyNonResourceAttributes) ||
			a.nonresource() == nil)
}

func (a *Attributes) IsReadOnly() bool {
	if a.IsResourceRequest() {
		return isVerbReadOnly(a.resource().Verb)
	}
	return isVerbReadOnly(a.nonresource().Verb)
}

func (a *Attributes) GetNamespace() string {
	if !a.IsResourceRequest() {
		// For some reason the RBAC authorizer still requests this even if
		// it's a non-resource request.
		return ""
	}
	return a.resource().Namespace
}

func (a *Attributes) GetResource() string {
	a.assertResourceRequest()
	return a.request.ResourceAttributes.Resource
}

func (a *Attributes) GetSubresource() string {
	return a.resource().Subresource
}

func (a *Attributes) GetName() string {
	return a.resource().Name
}

func (a *Attributes) GetAPIGroup() string {
	return a.resource().Group
}

func (a *Attributes) GetAPIVersion() string {
	return a.resource().Version
}

func (a *Attributes) GetPath() string {
	return a.nonresource().Path
}

// End of section implementing authorizer.Attributes.
//----------------------------------------------------------------------

// isVerbReadOnly checks whether verb corresponds to a REST verb for read-only
// operations.
func isVerbReadOnly(verb string) bool {
	switch verb {
	case "get":
		fallthrough
	case "list":
		fallthrough
	case "watch":
		return true
	default:
		return false
	}
}

// resource extracts the resource part of the attributes.  Panics if the
// attributes do not correspond to actual resource attributes.
func (a *Attributes) resource() *authz.ResourceAttributes {
	a.assertResourceRequest()
	return a.request.ResourceAttributes
}

// nonresource is similar to resource above, but for nonresource attributes.
func (a *Attributes) nonresource() *authz.NonResourceAttributes {
	return a.request.NonResourceAttributes
}

// assertResourceRequest panics if this is not a resource request.
func (a *Attributes) assertResourceRequest() {
	if !a.IsResourceRequest() {
		panic(fmt.Sprintf(
			"requested for resource in a non-resource request: %s", spew.Sdump(*a)))
	}
}

// convertFromSarExtra performs a vacuous type conversion from a map of string
// to slice of string into a map of, essentially, the same type.
func convertFromSarExtra(extra map[string]authz.ExtraValue) map[string][]string {
	if extra == nil {
		return nil
	}
	ret := map[string][]string{}
	for k, v := range extra {
		ret[k] = []string(v)
	}
	return ret
}
