package resource_quota

import (
	"fmt"

	"github.com/golang/glog"
	"github.com/google/stolos/pkg/api/policyhierarchy/v1"
	listers_v1 "github.com/google/stolos/pkg/client/listers/k8us/v1"
	"github.com/google/stolos/pkg/client/policyhierarchy"
	core_v1 "k8s.io/api/core/v1"
	api_errors "k8s.io/apimachinery/pkg/api/errors"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// UpsertStolosQuota describes the details of the action to upsert a stolos quota object.
type UpsertStolosQuota struct {
	namespace string
	quotaSpec v1.StolosResourceQuotaSpec

	stolosQuotaLister        listers_v1.StolosResourceQuotaLister
	policyHierarchiInterface policyhierarchy.Interface
}

// Execute executes the upsert by getting the existing version and either modifying it or creating a new one.
func (a *UpsertStolosQuota) Execute() error {
	existing, err := a.stolosQuotaLister.StolosResourceQuotas(a.namespace).Get(ResourceQuotaObjectName)

	if err != nil {
		if api_errors.IsNotFound(err) {
			glog.Infof("Creating stolos quota ns: %s, spec: %v", a.namespace, a.quotaSpec)
			_, err := a.policyHierarchiInterface.K8usV1().StolosResourceQuotas(a.namespace).Create(&v1.StolosResourceQuota{
				ObjectMeta: meta_v1.ObjectMeta{
					Name:   ResourceQuotaObjectName,
					Labels: PolicySpaceQuotaLabels,
				},
				Spec: a.quotaSpec,
			})
			return err
		}
		return err
	}

	if !specEqual(existing.Spec, a.quotaSpec) {
		glog.Infof("Updating stolos quota ns: %s, \nold spec: %v \nnew spec: %v",
			a.namespace, existing.Spec, a.quotaSpec)
		_, err := a.policyHierarchiInterface.K8usV1().StolosResourceQuotas(a.namespace).Update(&v1.StolosResourceQuota{
			ObjectMeta: meta_v1.ObjectMeta{
				Name:            ResourceQuotaObjectName,
				Labels:          PolicySpaceQuotaLabels,
				ResourceVersion: existing.ResourceVersion,
			},
			Spec: a.quotaSpec,
		})
		return err
	}
	return nil
}

// Returns true if the two stolos quota specs are equal. Reflection.deepEqual is not appropriate as
// the order of the items in the two maps may be different, and the same quantity can be expressed
// in multiple ways (i.e. 1.0 and 1 as strings)
func specEqual(left, right v1.StolosResourceQuotaSpec) bool {
	return resourceListEqual(left.Status.Hard, right.Status.Hard) &&
		resourceListEqual(left.Status.Used, right.Status.Used)
}

func resourceListEqual(left, right core_v1.ResourceList) bool {
	if len(left) != len(right) {
		return false
	}

	for resource, quantity := range left {
		if quantity.Cmp(right[resource]) != 0 {
			return false
		}
	}
	return true
}

func (a *UpsertStolosQuota) String() string {
	return fmt.Sprintf("stolos-quota-upsert.%s", a.namespace)
}
