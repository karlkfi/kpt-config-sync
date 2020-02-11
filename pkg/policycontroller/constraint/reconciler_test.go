package constraint

import (
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	nomosv1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/core"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestAnnotateConstraint(t *testing.T) {
	testCases := []struct {
		desc       string
		constraint unstructured.Unstructured
		want       map[string]string
	}{
		{
			"Constraint not yet processed",
			con().generation(5).build(),
			map[string]string{
				nomosv1.ResourceStatusUnreadyKey: `["Constraint has not been processed by PolicyController"]`,
			},
		},
		{
			"Constraint not yet enforced",
			con().generation(5).byPod(5, false).build(),
			map[string]string{
				nomosv1.ResourceStatusUnreadyKey: `["[0] PolicyController is not enforcing Constraint"]`,
			},
		},
		{
			"PolicyController has outdated version of Constraint",
			con().generation(5).byPod(4, true).build(),
			map[string]string{
				nomosv1.ResourceStatusUnreadyKey: `["[0] PolicyController has an outdated version of Constraint"]`,
			},
		},
		{
			"ConstraintTemplate has two errors",
			con().generation(5).byPod(5, true, "looks bad", "smells bad too").build(),
			map[string]string{
				nomosv1.ResourceStatusErrorsKey: `["[0] test-code: looks bad","[0] test-code: smells bad too"]`,
			},
		},
		{
			"ConstraintTemplate has error, but is out of date",
			con().generation(5).byPod(4, true, "looks bad").build(),
			map[string]string{
				nomosv1.ResourceStatusUnreadyKey: `["[0] PolicyController has an outdated version of Constraint"]`,
			},
		},
		{
			"Constraint is ready",
			con().generation(5).byPod(5, true).build(),
			nil,
		},
		{
			"Constraint had annotations previously, but is now ready",
			con().generation(5).annotateErrors("looks bad").annotateUnready("not yet").byPod(5, true).build(),
			map[string]string{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			annotateConstraint(tc.constraint)
			if diff := cmp.Diff(tc.want, tc.constraint.GetAnnotations()); diff != "" {
				t.Errorf("Incorrect annotations (-want +got):\n%s", diff)
			}
		})
	}
}

type conBuilder struct {
	unstructured.Unstructured
}

func con() *conBuilder {
	con := &conBuilder{
		Unstructured: unstructured.Unstructured{
			Object: map[string]interface{}{},
		},
	}
	con.SetGroupVersionKind(constraintGV.WithKind("TestConstraint"))
	return con
}

func (c *conBuilder) build() unstructured.Unstructured {
	return c.Unstructured
}

func (c *conBuilder) annotateErrors(msg string) *conBuilder {
	core.SetAnnotation(c, nomosv1.ResourceStatusErrorsKey, msg)
	return c
}

func (c *conBuilder) annotateUnready(msg string) *conBuilder {
	core.SetAnnotation(c, nomosv1.ResourceStatusUnreadyKey, msg)
	return c
}

func (c *conBuilder) generation(g int64) *conBuilder {
	c.SetGeneration(g)
	return c
}

func (c *conBuilder) byPod(generation int64, enforced bool, errMsgs ...string) *conBuilder {
	bps, saveChanges := newByPodStatus(c.Object)
	unstructured.SetNestedField(bps, generation, "observedGeneration")
	unstructured.SetNestedField(bps, enforced, "enforced")

	if len(errMsgs) > 0 {
		var statusErrs []interface{}
		for _, msg := range errMsgs {
			statusErrs = append(statusErrs, map[string]interface{}{
				"code":    "test-code",
				"message": msg,
			})
		}
		unstructured.SetNestedSlice(bps, statusErrs, "errors")
	}

	saveChanges()
	return c
}

// newByPodStatus appends a new byPodStatus to the byPod array of the given
// constraintTemplateStatus. It returns the byPodStatus as well as a function
// to call after the byPodStatus has been mutated to save changes. This function
// is necessary since SetNestedSlice() does a deep copy into the Unstructured.
func newByPodStatus(obj map[string]interface{}) (map[string]interface{}, func()) {
	bpArr, ok, _ := unstructured.NestedSlice(obj, "status", "byPod")
	if !ok {
		bpArr = []interface{}{}
	}

	byPodStatus := map[string]interface{}{}
	id := len(bpArr)
	unstructured.SetNestedField(byPodStatus, fmt.Sprintf("%d", id), "id")
	bpArr = append(bpArr, byPodStatus)

	return byPodStatus, func() {
		unstructured.SetNestedSlice(obj, bpArr, "status", "byPod")
	}
}
