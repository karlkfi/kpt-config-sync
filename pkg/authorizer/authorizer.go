/*
Copyright 2017 The Kubernetes Authors.
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

package authorizer

import (
	"fmt"

	"github.com/davecgh/go-spew/spew"
	"github.com/golang/glog"
	policyhierarchy "github.com/google/stolos/pkg/api/policyhierarchy/v1"
	"github.com/pkg/errors"
	authz "k8s.io/api/authorization/v1beta1"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
)

var (
	// TypeMeta is the SubjectAccessReview type meta.
	TypeMeta = meta.TypeMeta{
		Kind:       "SubjectAccessReview",
		APIVersion: "authorization.k8s.io/v1beta1",
	}
)

// Authorizer deals with the authorization mechanics.  Instantiate
// using a New() call below.
type Authorizer struct {
	// client is used to read the policy hierarchy.
	informer cache.SharedIndexInformer
}

// New creates an authorizer that watches the supplied informer for changes in
// the policy nodes structure.
func New(informer cache.SharedIndexInformer) *Authorizer {
	return &Authorizer{informer}
}

// policyRulesFor lists all policy rules that apply for the given 'namespace'.
// The returned PolicyNodeSpec index 0 is the policy node spec for the leaf
// namespace.  The last is for the root namespace.
func (a *Authorizer) policyRulesFor(
	namespace string) (*[]policyhierarchy.PolicyNodeSpec, error) {
	// Perhaps it is OK to return any policy node spec that has been
	// built so far.
	policies := make([]policyhierarchy.PolicyNodeSpec, 0)

	glog.V(5).Infof("PolicyNodes: %v", spew.Sdump(a.informer.GetStore().List()))

	// Follows a trail of namespaces starting from 'namespace', then
	// following the back-pointers to parents, up to the root PolicyNode.
	var err error
	nextNamespace := namespace
	for nextNamespace != "" {
		glog.V(4).Infof("policyRulesFor: resolving namespace: %v", nextNamespace)
		rawPolicyNode, exists, loopErr := a.informer.GetStore().
			GetByKey(nextNamespace)
		if loopErr != nil {
			err = errors.Wrapf(
				loopErr, "while resolving namespace: %v",
				nextNamespace)
			break
		}
		if !exists {
			err = errors.Errorf("partial policy rules, missing namespace: %v",
				nextNamespace)
			break
		}
		policyNode := rawPolicyNode.(*policyhierarchy.PolicyNode)
		policyNodeSpec := policyNode.Spec
		policies = append(policies, policyNodeSpec)
		nextNamespace = policyNodeSpec.Parent
	}
	glog.V(3).Infof("policyRulesFor: policies=%v, err=%v",
		spew.Sdump(policies), err)
	if err != nil {
		return &policies, errors.Wrapf(
			err, "while getting policy node: %s", namespace)
	}
	return &policies, nil
}

// Authorize verifies whether 'request' is allowed based on the current security
// context and the spec to be reviewed.  Returns SubjectAccessReviewStatus with
// the verdict.
func (a *Authorizer) Authorize(
	request *authz.SubjectAccessReviewSpec) *authz.SubjectAccessReviewStatus {
	attributes := request.ResourceAttributes
	if attributes == nil {
		return &authz.SubjectAccessReviewStatus{
			Allowed:         false,
			EvaluationError: "ResourceAttributes missing",
		}
	}

	_, err := a.policyRulesFor(request.ResourceAttributes.Namespace)
	if err != nil {
		// Oh, noes!  We could not get the policy rules, so we're
		// pretty much dead in the water.
		return &authz.SubjectAccessReviewStatus{
			Allowed: false,
			EvaluationError: fmt.Sprintf(
				"while getting hierarchical policy rules: %v",
				err),
		}
	}
	// TODO(fmil): Go and actually evaluate the access.
	return &authz.SubjectAccessReviewStatus{
		Allowed: true,
	}
}
