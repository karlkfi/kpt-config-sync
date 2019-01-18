package test

import (
	"context"
	"testing"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// AssertEventRecorded checks an event happend.
// The usage of this function would be cleaner if we could use the resource's
// TypeMeta as opposed to directly passing in the Kind. However, the test
// Reconciler clears the TypeMeta of the resource when Update is called. For
// now, just use the Kind passed in directly.
func AssertEventRecorded(t *testing.T, c *client.Client, kind string, om *metav1.ObjectMeta, reason string) {
	eventList := &v1.EventList{}
	if err := (*c).List(context.TODO(), &client.ListOptions{Namespace: om.Namespace}, eventList); err != nil {
		t.Fatalf("unable to list objects: %v", err)
	}
	for _, e := range eventList.Items {
		obj := &e.InvolvedObject
		if (obj.Kind == kind) && (obj.Namespace == om.Namespace) && (obj.Name == om.Name) {
			if e.Reason == reason {
				return
			}
		}
	}
	t.Errorf("event with reason '%v' not recorded for %v %v/%v", reason, kind, om.Namespace, om.Name)
}
