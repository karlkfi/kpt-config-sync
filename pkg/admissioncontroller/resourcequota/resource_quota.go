// Reviewed by sunilarora
/*
Copyright 2017 The Nomos Authors.
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

// Package resourcequota is the Resource Quota admission controller which will do a hierarchical
// evaluation of quota objects to ensure quota is not being violated whenever resources get
// created/modified along the hierarchy of namespaces.
package resourcequota

import (
	"strconv"
	"time"

	informerspolicynodev1 "github.com/google/nomos/clientgen/informers/externalversions/policyhierarchy/v1"
	"github.com/google/nomos/pkg/admissioncontroller"
	"github.com/google/nomos/pkg/resourcequota"
	admissionv1beta1 "k8s.io/api/admission/v1beta1"
	core_v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	informerscorev1 "k8s.io/client-go/informers/core/v1"
	"k8s.io/kubernetes/pkg/quota"
	"k8s.io/kubernetes/pkg/quota/generic"
	quotainstall "k8s.io/kubernetes/pkg/quota/install"
)

// Admitter is the struct for the resource quota admitter
type Admitter struct {
	policyNodeInformer    informerspolicynodev1.PolicyNodeInformer
	resourceQuotaInformer informerscorev1.ResourceQuotaInformer
	quotaRegistry         quota.Registry
	decoder               runtime.Decoder
}

var _ admissioncontroller.Admitter = (*Admitter)(nil)

// NewAdmitter returns the resource quota admitter
func NewAdmitter(policyNodeInformer informerspolicynodev1.PolicyNodeInformer,
	resourceQuotaInformer informerscorev1.ResourceQuotaInformer) admissioncontroller.Admitter {
	quotaConfiguration := quotainstall.NewQuotaConfigurationForAdmission()
	quotaRegistry := generic.NewRegistry(quotaConfiguration.Evaluators())

	// Decoder. Right now, only v1 types are needed as they are the only ones being monitored
	scheme := runtime.NewScheme()
	if err := core_v1.AddToScheme(scheme); err != nil {
		panic(err)
	}
	decoder := serializer.NewCodecFactory(scheme).UniversalDeserializer()

	return &Admitter{
		policyNodeInformer:    policyNodeInformer,
		resourceQuotaInformer: resourceQuotaInformer,
		quotaRegistry:         quotaRegistry,
		decoder:               decoder,
	}
}

// Admit decides whether to admit a request.
func (r *Admitter) Admit(review admissionv1beta1.AdmissionReview) *admissionv1beta1.AdmissionResponse {
	if review.Request == nil {
		return &admissionv1beta1.AdmissionResponse{
			Allowed: true,
		}
	}
	start := time.Now()
	resp := r.internalAdmit(review)
	elapsed := time.Since(start).Seconds()
	admissioncontroller.Metrics.AdmitDuration.WithLabelValues("resource_quota", review.Request.Namespace, strconv.FormatBool(resp.Allowed)).Observe(elapsed)
	return resp
}

func (r *Admitter) internalAdmit(review admissionv1beta1.AdmissionReview) *admissionv1beta1.AdmissionResponse {
	cache, err := resourcequota.NewHierarchicalQuotaCache(r.policyNodeInformer, r.resourceQuotaInformer)
	counter := admissioncontroller.Metrics.ErrorTotal.WithLabelValues("resource_quota", review.Request.Namespace)
	if err != nil {
		counter.Inc()
		return admissioncontroller.Deny(metav1.StatusReasonInternalError, err)
	}
	newUsage, err := r.getNewUsage(*review.Request)
	if err != nil {
		counter.Inc()
		return admissioncontroller.Deny(metav1.StatusReasonInternalError, err)
	}
	admitError := cache.Admit(review.Request.Namespace, newUsage)
	if admitError != nil {
		return admissioncontroller.Deny(metav1.StatusReasonForbidden, admitError)
	}
	return &admissionv1beta1.AdmissionResponse{
		Allowed: true,
	}
}

// getNewUsage returns the resource usage that would result from the given request.
func (r *Admitter) getNewUsage(request admissionv1beta1.AdmissionRequest) (core_v1.ResourceList, error) {
	v1NewUsage := core_v1.ResourceList{}
	attributes := admissioncontroller.GetAttributes(r.decoder, request)
	evaluator := r.quotaRegistry.Get(attributes.GetResource().GroupResource())

	if evaluator != nil && evaluator.Handles(attributes) {
		newUsage, err := evaluator.Usage(attributes.GetObject())
		if err != nil {
			return nil, err
		}
		for key, val := range newUsage {
			v1NewUsage[core_v1.ResourceName(key)] = val
		}
	}
	return v1NewUsage, nil
}
