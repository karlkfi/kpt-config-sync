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

	"github.com/golang/glog"
	"github.com/google/stolos/pkg/api/policyhierarchy/v1"
	k8usv1 "github.com/google/stolos/pkg/client/policyhierarchy/typed/k8us/v1"
	apierrors "github.com/pkg/errors"
	authz "k8s.io/api/authorization/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	// TypeMeta is the SubjectAccessReview type meta.
	TypeMeta = metav1.TypeMeta{
		Kind:       "SubjectAccessReview",
		APIVersion: "authorization.k8s.io/v1beta1",
	}
)

// Authorizer deals with the authorization mechanics.  Instantiate
// using a New() call below.
type Authorizer struct {
	// client is used to read the policy hierarchy.
	client k8usv1.K8usV1Interface
}

// New creates a new authorizer based on the supplied policy hierarchy 'client'.
func New(client k8usv1.K8usV1Interface) *Authorizer {
	return &Authorizer{client}
}

// policyRulesFor lists all policy rules that apply for the given 'namespace'.
// The returned PolicyNodeSpec index 0 is the policy node spec for the leaf
// namespace.  The last is for the root namespace.
// TODO(fmil): Fix the approach below: this should rely on examining Namespace;
// but in that case I don't know what the resulting objects for enforcement
// will be.
func (a *Authorizer) policyRulesFor(
	namespace string) (*[]v1.PolicyNodeSpec, error) {
	// Perhaps it is OK to return any policy node spec that has been
	// built so far.
	policies := make([]v1.PolicyNodeSpec, 0)
	var err error
	resolvedNamespace := namespace
	for resolvedNamespace != "" || err != nil {
		glog.V(1).Infof("Getting namespace: '%v'", resolvedNamespace)
		result, loopErr := a.client.PolicyNodes().Get(
			resolvedNamespace, metav1.GetOptions{})
		if loopErr != nil {
			glog.V(2).Infof("while resolving: %v: %v",
				resolvedNamespace, err)
			err = loopErr
			break
		}
		spec := result.Spec
		policies = append(policies, spec)
		// For the next iteration.
		resolvedNamespace = spec.Parent
	}
	if err != nil {
		return &policies, apierrors.Wrapf(
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
