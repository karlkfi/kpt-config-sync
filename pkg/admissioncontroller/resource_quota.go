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

// This is the Resource Quota admission controller which will do a hierarchical evaluation of quota objects
// to ensure quota is not being violated whenever resources get created/modified along the hierarchy of namespaces.
package admissioncontroller

import (
	"strconv"
	"time"

	informerspolicynodev1 "github.com/google/nomos/pkg/client/informers/externalversions/policyhierarchy/v1"
	"github.com/google/nomos/pkg/resourcequota"
	"github.com/prometheus/client_golang/prometheus"
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

// Prometheus metrics
var (
	admitDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Help:      "Quota admission duration distributions",
			Namespace: "nomos",
			Subsystem: "quota_admission",
			Name:      "action_duration_seconds",
			Buckets:   []float64{.001, .0025, .005, .01, .025, .05, .1, .25, .5, 1, 2.5},
		},
		[]string{"namespace", "allowed"},
	)
	errTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Help:      "Total internal errors that occurred when reviewing quota requests",
			Namespace: "nomos",
			Subsystem: "quota_admission",
			Name:      "error_total",
		},
		[]string{"namespace"},
	)
)

func init() {
	prometheus.MustRegister(admitDuration)
	prometheus.MustRegister(errTotal)
}

type ResourceQuotaAdmitter struct {
	policyNodeInformer    informerspolicynodev1.PolicyNodeInformer
	resourceQuotaInformer informerscorev1.ResourceQuotaInformer
	quotaRegistry         quota.Registry
	decoder               runtime.Decoder
}

var _ Admitter = (*ResourceQuotaAdmitter)(nil)

func NewResourceQuotaAdmitter(policyNodeInformer informerspolicynodev1.PolicyNodeInformer,
	resourceQuotaInformer informerscorev1.ResourceQuotaInformer) Admitter {
	quotaConfiguration := quotainstall.NewQuotaConfigurationForAdmission()
	quotaRegistry := generic.NewRegistry(quotaConfiguration.Evaluators())

	// Decoder. Right now, only v1 types are needed as they are the only ones being monitored
	scheme := runtime.NewScheme()
	core_v1.AddToScheme(scheme)

	decoder := serializer.NewCodecFactory(scheme).UniversalDeserializer()

	return &ResourceQuotaAdmitter{
		policyNodeInformer:    policyNodeInformer,
		resourceQuotaInformer: resourceQuotaInformer,
		quotaRegistry:         quotaRegistry,
		decoder:               decoder,
	}
}

// Decides whether to admit a request.
func (r *ResourceQuotaAdmitter) Admit(review admissionv1beta1.AdmissionReview) *admissionv1beta1.AdmissionResponse {
	if review.Request == nil {
		return &admissionv1beta1.AdmissionResponse{
			Allowed: true,
		}
	}
	start := time.Now()
	resp := r.internalAdmit(review)
	elapsed := time.Since(start).Seconds()
	admitDuration.WithLabelValues(review.Request.Namespace, strconv.FormatBool(resp.Allowed)).Observe(elapsed)
	return resp
}

func (r *ResourceQuotaAdmitter) internalAdmit(review admissionv1beta1.AdmissionReview) *admissionv1beta1.AdmissionResponse {
	cache, err := resourcequota.NewHierarchicalQuotaCache(r.policyNodeInformer, r.resourceQuotaInformer)
	if err != nil {
		return internalErrorDeny(err, review.Request.Namespace)
	}
	newUsage, err := r.getNewUsage(*review.Request)
	if err != nil {
		return internalErrorDeny(err, review.Request.Namespace)
	}
	admitError := cache.Admit(review.Request.Namespace, newUsage)
	if admitError != nil {
		return &admissionv1beta1.AdmissionResponse{
			Allowed: false,
			Result: &metav1.Status{
				Message: admitError.Error(),
				Reason:  metav1.StatusReason(metav1.StatusReasonForbidden),
			},
		}
	}
	return &admissionv1beta1.AdmissionResponse{
		Allowed: true,
	}
}

// getNewUsage returns the resource usage that would result from the given request.
func (r *ResourceQuotaAdmitter) getNewUsage(request admissionv1beta1.AdmissionRequest) (core_v1.ResourceList, error) {
	v1NewUsage := core_v1.ResourceList{}
	attributes := getAttributes(r.decoder, request)
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

func internalErrorDeny(err error, namespace string) *admissionv1beta1.AdmissionResponse {
	errTotal.WithLabelValues(namespace).Inc()
	return &admissionv1beta1.AdmissionResponse{
		Allowed: false,
		Result: &metav1.Status{
			Message: err.Error(),
			Reason:  metav1.StatusReason(metav1.StatusReasonInternalError),
		},
	}
}
