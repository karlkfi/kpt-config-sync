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

// This is the Policy Node admission controller. It ensures that CUD operations on PolicyNodes
// keep the structure of PolicyNodes in a consistent tree-like state.
package policynode

import (
	"strconv"
	"time"

	"github.com/golang/glog"
	"github.com/google/nomos/pkg/admissioncontroller"
	policyhierarchy_v1 "github.com/google/nomos/pkg/api/policyhierarchy/v1"
	informerspolicynodev1 "github.com/google/nomos/pkg/client/informers/externalversions/policyhierarchy/v1"
	"github.com/google/nomos/pkg/util/policynode/validator"
	"github.com/prometheus/client_golang/prometheus"
	admissionv1beta1 "k8s.io/api/admission/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apiserver/pkg/admission"
)

// This is the Policy Node Admission Controller which verifies that CUD operations on PolicyNodes
// are valid.
type Admitter struct {
	policyNodeInformer informerspolicynodev1.PolicyNodeInformer
	decoder            runtime.Decoder
	admitDuration      *prometheus.HistogramVec
	errTotal           *prometheus.CounterVec
}

var _ admissioncontroller.Admitter = (*Admitter)(nil)

func NewAdmitter(policyNodeInformer informerspolicynodev1.PolicyNodeInformer) admissioncontroller.Admitter {
	// Decoder. Right now, only v1 types are needed as they are the only ones being monitored
	scheme := runtime.NewScheme()
	policyhierarchy_v1.AddToScheme(scheme)

	decoder := serializer.NewCodecFactory(scheme).UniversalDeserializer()

	// Prometheus metrics
	admitDuration := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Help:      "Policy Node admission duration distributions",
			Namespace: "nomos",
			Subsystem: "policy_node_admission",
			Name:      "action_duration_seconds",
			Buckets:   []float64{.001, .0025, .005, .01, .025, .05, .1, .25, .5, 1, 2.5},
		},
		[]string{"namespace", "allowed"},
	)
	errTotal := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Help:      "Total internal errors that occurred when reviewing policy node requests",
			Namespace: "nomos",
			Subsystem: "policy_node_admission",
			Name:      "error_total",
		},
		[]string{"namespace"},
	)
	prometheus.MustRegister(admitDuration)
	prometheus.MustRegister(errTotal)

	return &Admitter{
		policyNodeInformer: policyNodeInformer,
		decoder:            decoder,
		admitDuration:      admitDuration,
		errTotal:           errTotal,
	}
}

// Decides whether or not to admit a request.
func (p *Admitter) Admit(review admissionv1beta1.AdmissionReview) *admissionv1beta1.AdmissionResponse {
	if review.Request == nil {
		return &admissionv1beta1.AdmissionResponse{
			Allowed: true,
		}
	}
	start := time.Now()
	resp := p.internalAdmit(review)
	elapsed := time.Since(start).Seconds()
	p.admitDuration.WithLabelValues(review.Request.Namespace, strconv.FormatBool(resp.Allowed)).Observe(elapsed)
	return resp
}

func (p *Admitter) internalAdmit(
	review admissionv1beta1.AdmissionReview) *admissionv1beta1.AdmissionResponse {
	policyNodes, err := p.policyNodeInformer.Lister().List(labels.Everything())
	if err != nil {
		return admissioncontroller.InternalErrorDeny(p.errTotal, err, review.Request.Namespace)
	}

	// TODO(sbochins): A way for the user to toggle multiple roots, orphan adds checks.
	validator := validator.From(policyNodes...)
	validator.AllowMultipleRoots = true
	validator.AllowOrphanAdds = true
	if err := p.validateAdmit(validator, *review.Request); err != nil {
		return &admissionv1beta1.AdmissionResponse{
			Allowed: false,
			Result: &metav1.Status{
				Message: err.Error(),
				Reason:  metav1.StatusReason(metav1.StatusReasonForbidden),
			},
		}
	}
	return &admissionv1beta1.AdmissionResponse{
		Allowed: true,
	}
}

func (p *Admitter) validateAdmit(
	validator *validator.Validator, request admissionv1beta1.AdmissionRequest) error {
	attributes := admissioncontroller.GetAttributes(p.decoder, request)

	var policyNode *policyhierarchy_v1.PolicyNode
	if obj := attributes.GetObject(); obj == nil {
		if request.Name == "" {
			// This should never happen. The request does not have an object and does not have the name of an object.
			glog.Warningf("Request with no object or name used in policy node admission controller: %v", request)
			return nil
		}
		policyNode = &policyhierarchy_v1.PolicyNode{
			ObjectMeta: metav1.ObjectMeta{
				Name: request.Name,
			},
		}
	} else {
		var ok bool
		policyNode, ok = obj.(*policyhierarchy_v1.PolicyNode)
		if !ok {
			// This should never happen. We're only checking for PolicyNodes in this controller.
			// Just accept anything that isn't a PolicyNode.
			glog.Warningf("Request operating on non policy node used in policy node admission controller: %v", request)
			return nil
		}
	}

	operation := attributes.GetOperation()
	var err error
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
		glog.Warningf("Request with operation, %s, used in policy node admission controller: %v", operation, request)
	}

	if err != nil {
		return err
	}

	return validator.Validate()
}
