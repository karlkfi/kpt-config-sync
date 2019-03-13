package status

import (
	"testing"

	"github.com/pkg/errors"
)

var errFoo = UndocumentedError("foo")
var errBar = apiServerError{errors.New("bar")}
var errBaz = UndocumentedError("baz")

var errFooRaw = errors.New("raw foo")
var errBarRaw = errors.New("raw bar")

func TestErrorBuilder(t *testing.T) {
	for _, tc := range []struct {
		name   string
		errors []error
		want   *MultiError
	}{
		{
			"build golang errors",
			[]error{errFooRaw, errBarRaw},
			&MultiError{errs: []Error{UndocumentedWrapf(errFooRaw, ""), UndocumentedWrapf(errBarRaw, "")}},
		},
		{
			"build status Errors",
			[]error{errFoo, errBar},
			&MultiError{errs: []Error{errFoo, errBar}},
		},
		{
			"build nil errors",
			[]error{nil, nil},
			nil,
		},
		{
			"build mixed errors",
			[]error{errBaz, nil, errFooRaw},
			&MultiError{errs: []Error{errBaz, UndocumentedWrapf(errFooRaw, "")}},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var builder ErrorBuilder
			for _, err := range tc.errors {
				builder.Add(err)
			}
			got := builder.Build()

			if tc.want == nil {
				if builder.HasErrors() {
					t.Errorf("got %d errors; want 0 errors", builder.Len())
				}
				if got != nil {
					t.Errorf("got %v; want nil", got)
				}
			} else if got == nil {
				t.Errorf("got nil; want %v", tc.want)
			} else {
				wantErrorLen := len(tc.want.Errors())

				if !builder.HasErrors() || builder.Len() != wantErrorLen {
					t.Errorf("got %d errors; want %d errors", builder.Len(), wantErrorLen)
				}
				if got.Error() != tc.want.Error() {
					t.Errorf("got %v; want %v", got, tc.want)
				}
			}
		})
	}
}

func TestMultiErrorFrom(t *testing.T) {
	for _, tc := range []struct {
		name   string
		errors []Error
		want   *MultiError
	}{
		{
			"from one error",
			[]Error{errFoo},
			&MultiError{errs: []Error{errFoo}},
		},
		{
			"from multiple errors",
			[]Error{errFoo, errBar, errBaz},
			&MultiError{errs: []Error{errFoo, errBar, errBaz}},
		},
		{
			"from no errors",
			[]Error{},
			nil,
		},
		{
			"from nil error",
			[]Error{nil},
			nil,
		},
		{
			"from mixed errors",
			[]Error{errFoo, nil, errBar},
			&MultiError{errs: []Error{errFoo, errBar}},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := From(tc.errors...)

			if tc.want == nil {
				if got != nil {
					t.Errorf("got %v; want nil", got)
				}
			} else if got == nil {
				t.Errorf("got nil; want %v", tc.want)
			} else if got.Error() != tc.want.Error() {
				t.Errorf("got %v; want %v", got, tc.want)
			}
		})
	}
}
