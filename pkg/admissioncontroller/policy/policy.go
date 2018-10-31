/*
Copyright 2018 The Nomos Authors.
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

// Package policy contains the policy admission controller. It ensures that CUD operations
// on PolicyNodes keep the structure of PolicyNodes in a consistent tree-like state. It also
// ensures the ClusterPolicy is valid.
package policy

import (
	"strconv"
	"time"

	"github.com/golang/glog"
	policyinformer "github.com/google/nomos/clientgen/informer/policyhierarchy/v1"
	"github.com/google/nomos/pkg/admissioncontroller"
	"github.com/google/nomos/pkg/api/policyhierarchy/v1"
	"github.com/google/nomos/pkg/util/clusterpolicy"
	"github.com/google/nomos/pkg/util/policynode/validator"
	admissionv1beta1 "k8s.io/api/admission/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apiserver/pkg/admission"
)

// Admitter is the Policy Node Admission Controller which verifies that CUD operations on PolicyNodes
// and ClusterPolicies are valid.
type Admitter struct {
	policyNodeInformer policyinformer.PolicyNodeInformer
	decoder            runtime.Decoder
}

var _ admissioncontroller.Admitter = (*Admitter)(nil)

// NewAdmitter returns the policy admitter
func NewAdmitter(policyNodeInformer policyinformer.PolicyNodeInformer) admissioncontroller.Admitter {
	// Decoder. Right now, only v1 types are needed as they are the only ones being monitored
	scheme := runtime.NewScheme()
	if err := v1.AddToScheme(scheme); err != nil {
		panic(err)
	}

	decoder := serializer.NewCodecFactory(scheme).UniversalDeserializer()

	return &Admitter{
		policyNodeInformer: policyNodeInformer,
		decoder:            decoder,
	}
}

// Admit decides whether or not to admit a request.
func (p *Admitter) Admit(review admissionv1beta1.AdmissionReview) *admissionv1beta1.AdmissionResponse {
	if review.Request == nil {
		return &admissionv1beta1.AdmissionResponse{
			Allowed: true,
		}
	}
	start := time.Now()
	resp := p.internalAdmit(review)
	elapsed := time.Since(start).Seconds()
	admissioncontroller.Metrics.AdmitDuration.WithLabelValues("policy", review.Request.Namespace, strconv.FormatBool(resp.Allowed)).Observe(elapsed)
	return resp
}

func (p *Admitter) internalAdmit(
	review admissionv1beta1.AdmissionReview) *admissionv1beta1.AdmissionResponse {
	switch review.Request.Kind.Kind {
	case "ClusterPolicy":
		if errResp := p.clusterPolicyAdmit(*review.Request); errResp != nil {
			return errResp
		}
	case "PolicyNode":
		if errResp := p.policyNodeAdmit(*review.Request); errResp != nil {
			return errResp
		}
	default:
		// This should never happen. We're only checking for PolicyNodes and ClusterPolicies in this controller.
		// Just accept anything that isn't a Nomos policy resource.
		glog.Warningf("Request operating on non nomos policy resource used in policy admission controller: %v", review.Request)
	}
	return &admissionv1beta1.AdmissionResponse{
		Allowed: true,
	}
}

func (p *Admitter) clusterPolicyAdmit(request admissionv1beta1.AdmissionRequest) *admissionv1beta1.AdmissionResponse {
	attributes := admissioncontroller.GetAttributes(p.decoder, request)

	var clusterPolicy *v1.ClusterPolicy
	if obj := attributes.GetObject(); obj == nil {
		if attributes.GetName() == "" {
			// This should never happen. The request does not have an object and does not have the name of an object.
			glog.Warningf("Request with no object or name used in policy admission controller: %v", request)
			return nil
		}
		clusterPolicy = &v1.ClusterPolicy{
			ObjectMeta: metav1.ObjectMeta{
				Name: attributes.GetName(),
			},
		}
	} else {
		var ok bool
		clusterPolicy, ok = obj.(*v1.ClusterPolicy)
		if !ok {
			// This should never happen. The Kind of the resource is PolicyNode, but the object isn't a PolicyNode.
			glog.Warningf("Request specifies Kind ClusterPolicy, but a non ClusterPolicy object was included: %v", request)
			return nil
		}
	}

	operation := attributes.GetOperation()
	var err error
	switch operation {
	case admission.Create, admission.Update:
		if err = clusterpolicy.Validate(clusterPolicy); err != nil {
			return admissioncontroller.Deny(metav1.StatusReasonForbidden, err)
		}
	case admission.Delete:
		// Don't do name checking. Allow the user to delete clusterpolicies that aren't using the system generated name,
		// if they somehow ended up there.
	default:
		// This should never happen. We're only checking CUD operations in this controller.
		// Just accept operations we aren't concerned with.
		glog.Warningf("Request for clusterpolicy with operation, %s, used in policy admission controller: %v", operation,
			request)
	}

	return nil
}

func (p *Admitter) policyNodeAdmit(request admissionv1beta1.AdmissionRequest) *admissionv1beta1.AdmissionResponse {
	attributes := admissioncontroller.GetAttributes(p.decoder, request)

	var policyNode *v1.PolicyNode
	if obj := attributes.GetObject(); obj == nil {
		if attributes.GetName() == "" {
			// This should never happen. The request does not have an object and does not have the name of an object.
			glog.Warningf("Request with no object or name used in policy admission controller: %v", request)
			return nil
		}
		policyNode = &v1.PolicyNode{
			ObjectMeta: metav1.ObjectMeta{
				Name: attributes.GetName(),
			},
		}
	} else {
		var ok bool
		policyNode, ok = obj.(*v1.PolicyNode)
		if !ok {
			// This should never happen. The Kind of the resource is PolicyNode, but the object isn't a PolicyNode.
			glog.Warningf("Request specifies Kind PolicyNode, but a non PolicyNode object was included: %v", request)
			return nil
		}
	}

	policyNodes, err := p.policyNodeInformer.Lister().List(labels.Everything())
	if err != nil {
		admissioncontroller.Metrics.ErrorTotal.WithLabelValues("policy", request.Namespace).Inc()
		return admissioncontroller.Deny(metav1.StatusReasonInternalError, err)
	}
	// TODO(77731524): A way for the user to toggle multiple roots, orphan adds checks.
	validator := validator.From(policyNodes...)
	validator.AllowMultipleRoots = true

	operation := attributes.GetOperation()
	switch operation {
	case admission.Create:
		err = validator.Add(policyNode)
	case admission.Delete:
		err = validator.Remove(policyNode)
	case admission.Update:
		err = validator.Update(policyNode)
	default:
		// This should never happen. We're only checking CUD operations in this controller.
		// Just accept operations we aren't concerned with.
		glog.Warningf("Request for policynode with operation, %s, used in policy admission controller: %v", operation,
			request)
	}
	if err != nil {
		return admissioncontroller.Deny(metav1.StatusReasonForbidden, err)
	}

	if err := validator.Validate(); err != nil {
		return admissioncontroller.Deny(metav1.StatusReasonForbidden, err)
	}
	return nil
}
