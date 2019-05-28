// Package resourcequota is the Resource Quota admission controller which will do a hierarchical
// evaluation of quota objects to ensure quota is not being violated whenever resources get
// created/modified along the hierarchy of namespaces.
package resourcequota

import (
	"strconv"
	"time"

	informersv1 "github.com/google/nomos/clientgen/informer/configmanagement/v1"
	"github.com/google/nomos/pkg/admissioncontroller"
	"github.com/google/nomos/pkg/resourcequota"
	admissionv1beta1 "k8s.io/api/admission/v1beta1"
	corev1 "k8s.io/api/core/v1"
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
	resourceQuotaInformer     informerscorev1.ResourceQuotaInformer
	hierarchicalQuotaInformer informersv1.HierarchicalQuotaInformer
	quotaRegistry             quota.Registry
	decoder                   runtime.Decoder
}

var _ admissioncontroller.Admitter = (*Admitter)(nil)

// NewAdmitter returns the resource quota admitter
func NewAdmitter(
	resourceQuotaInformer informerscorev1.ResourceQuotaInformer,
	hierarchicalQuotaInformer informersv1.HierarchicalQuotaInformer) admissioncontroller.Admitter {
	quotaConfiguration := quotainstall.NewQuotaConfigurationForAdmission()
	quotaRegistry := generic.NewRegistry(quotaConfiguration.Evaluators())

	// Decoder. Right now, only v1 types are needed as they are the only ones being monitored
	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		panic(err)
	}
	decoder := serializer.NewCodecFactory(scheme).UniversalDeserializer()

	return &Admitter{
		resourceQuotaInformer:     resourceQuotaInformer,
		hierarchicalQuotaInformer: hierarchicalQuotaInformer,
		quotaRegistry:             quotaRegistry,
		decoder:                   decoder,
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
	admissioncontroller.Metrics.AdmitDuration.WithLabelValues(strconv.FormatBool(resp.Allowed)).Observe(elapsed)
	return resp
}

func (r *Admitter) internalAdmit(review admissionv1beta1.AdmissionReview) *admissionv1beta1.AdmissionResponse {
	cache, err := resourcequota.NewHierarchicalQuotaCache(r.resourceQuotaInformer, r.hierarchicalQuotaInformer)
	if err != nil {
		admissioncontroller.Metrics.ErrorTotal.Inc()
		return admissioncontroller.Deny(metav1.StatusReasonInternalError, err)
	}
	newUsage, err := r.getNewUsage(*review.Request)
	if err != nil {
		admissioncontroller.Metrics.ErrorTotal.Inc()
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
func (r *Admitter) getNewUsage(request admissionv1beta1.AdmissionRequest) (corev1.ResourceList, error) {
	v1NewUsage := corev1.ResourceList{}
	attributes := admissioncontroller.GetAttributes(r.decoder, request)
	evaluator := r.quotaRegistry.Get(attributes.GetResource().GroupResource())

	if evaluator != nil && evaluator.Handles(attributes) {
		newUsage, err := evaluator.Usage(attributes.GetObject())
		if err != nil {
			return nil, err
		}
		for key, val := range newUsage {
			v1NewUsage[corev1.ResourceName(key)] = val
		}
	}
	return v1NewUsage, nil
}
