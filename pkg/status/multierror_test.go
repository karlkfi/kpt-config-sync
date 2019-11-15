package status

import (
	"testing"

	"github.com/pkg/errors"
)

var errFoo = UndocumentedError("foo")
var errBar = APIServerError(errors.New("bar"), "qux")
var errBaz = UndocumentedError("baz")

var errFooRaw = errors.New("raw foo")
var errBarRaw = errors.New("raw bar")

func TestErrorBuilder(t *testing.T) {
	for _, tc := range []struct {
		name   string
		errors []error
		want   MultiError
	}{
		{
			"build golang errors",
			[]error{errFooRaw, errBarRaw},
			&multiError{errs: []Error{undocumented(errFooRaw), undocumented(errBarRaw)}},
		},
		{
			"build status Errors",
			[]error{errFoo, errBar},
			&multiError{errs: []Error{errFoo, errBar}},
		},
		{
			"build nil errors",
			[]error{nil, nil},
			nil,
		},
		{
			"build mixed errors",
			[]error{errBaz, nil, errFooRaw},
			&multiError{errs: []Error{errBaz, undocumented(errFooRaw)}},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var errs MultiError
			for _, err := range tc.errors {
				errs = Append(errs, err)
			}

			if tc.want == nil {
				if errs != nil {
					t.Errorf("got %d errors; want 0 errors", len(errs.Errors()))
				}
				if errs != nil {
					t.Errorf("got %v; want nil", errs)
				}
			} else if errs == nil {
				t.Errorf("got nil; want %v", tc.want)
			} else {
				wantErrorLen := len(tc.want.Errors())

				if len(errs.Errors()) != wantErrorLen {
					t.Errorf("got %d errors; want %d errors", len(errs.Errors()), wantErrorLen)
				}
				if errs.Error() != tc.want.Error() {
					t.Errorf("got %v; want %v", errs, tc.want)
				}
			}
		})
	}
}
