package policycontroller

import (
	"github.com/google/nomos/pkg/policycontroller/constraint"
	"github.com/google/nomos/pkg/policycontroller/constrainttemplate"
	"github.com/google/nomos/pkg/util/watch"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

type builder struct{}

var _ watch.ControllerBuilder = &builder{}

// StartControllers starts a new constraint controller for each of the specified
// constraint GVKs. It also starts the controller for ConstraintTemplates if
// their CRD is present.
func (b *builder) StartControllers(mgr manager.Manager, gvks map[schema.GroupVersionKind]bool, _ metav1.Time) error {
	ctGVK := constrainttemplate.GVK.String()
	for gvk := range gvks {
		if gvk.String() == ctGVK {
			if err := constrainttemplate.AddController(mgr); err != nil {
				return errors.Wrap(err, "controller for ConstraintTemplate")
			}
		} else if err := constraint.AddController(mgr, gvk.Kind); err != nil {
			return errors.Wrapf(err, "controller for %s", gvk.String())
		}
	}
	return nil
}
