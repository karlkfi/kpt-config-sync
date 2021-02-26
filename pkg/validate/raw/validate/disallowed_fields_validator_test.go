package validate

import (
	"errors"
	"testing"
	"time"

	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/validation/syntax"
	"github.com/google/nomos/pkg/importer/id"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/testing/fake"
	"github.com/google/nomos/pkg/validate/objects"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestDisallowedFields(t *testing.T) {
	testCases := []struct {
		name     string
		objs     *objects.Raw
		wantErrs status.MultiError
	}{
		{
			name: "Deployment with allowed fields passes",
			objs: &objects.Raw{
				Objects: []ast.FileObject{
					fake.Deployment("hello"),
				},
			},
		},
		{
			name: "Deployment with disallowed fields fails",
			objs: &objects.Raw{
				Objects: []ast.FileObject{
					fake.Deployment("hello",
						core.OwnerReference([]metav1.OwnerReference{{}}),
						core.SelfLink("this-is-me"),
						core.UID("my-uid"),
						core.ResourceVersion("123456"),
						core.Generation(123456),
						core.CreationTimeStamp(metav1.NewTime(time.Now())),
						core.DeletionTimestamp(metav1.NewTime(time.Now())),
						core.DeletionGracePeriod(654321),
					),
				},
			},
			wantErrs: status.Append(nil,
				syntax.IllegalFieldsInConfigError(fake.Deployment("hello"), id.OwnerReference),
				syntax.IllegalFieldsInConfigError(fake.Deployment("hello"), id.SelfLink),
				syntax.IllegalFieldsInConfigError(fake.Deployment("hello"), id.UID),
				syntax.IllegalFieldsInConfigError(fake.Deployment("hello"), id.ResourceVersion),
				syntax.IllegalFieldsInConfigError(fake.Deployment("hello"), id.Generation),
				syntax.IllegalFieldsInConfigError(fake.Deployment("hello"), id.CreationTimestamp),
				syntax.IllegalFieldsInConfigError(fake.Deployment("hello"), id.DeletionTimestamp),
				syntax.IllegalFieldsInConfigError(fake.Deployment("hello"), id.DeletionGracePeriodSeconds),
			),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			errs := DisallowedFields(tc.objs)
			if !errors.Is(errs, tc.wantErrs) {
				t.Errorf("got DisallowedFields() error %v, want %v", errs, tc.wantErrs)
			}
		})
	}
}
