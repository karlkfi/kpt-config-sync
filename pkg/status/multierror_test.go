package status

import (
	"fmt"
	"strings"
	"testing"

	"github.com/pkg/errors"
)

var errFoo = UndocumentedError("foo")
var errBar = APIServerError(errors.New("bar"), "qux")
var errBaz = UndocumentedError("baz")

var errFooRaw = errors.New("raw foo")
var errBarRaw = errors.New("raw bar")

func multilineError() string {
	b := strings.Builder{}
	b.WriteString(fmt.Sprintf("%s\n\n\n", "2 error(s)"))
	b.WriteString(fmt.Sprintf("%s\n\n", "[1] KNV9999: raw bar"))
	b.WriteString(fmt.Sprintf("%s\n\n\n", "For more information, see https://cloud.google.com/anthos-config-management/docs/reference/errors#knv9999"))
	b.WriteString(fmt.Sprintf("%s\n\n", "[2] KNV9999: raw foo"))
	b.WriteString(fmt.Sprintf("%s\n", "For more information, see https://cloud.google.com/anthos-config-management/docs/reference/errors#knv9999"))
	return b.String()
}

func singlelineError() string {
	b := strings.Builder{}
	b.WriteString(fmt.Sprintf("%s\n", "1 error(s) "))
	b.WriteString("[1] KNV9999: raw bar  ")
	b.WriteString("For more information, see https://cloud.google.com/anthos-config-management/docs/reference/errors#knv9999 ")
	return b.String()
}

func TestAppend(t *testing.T) {
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
		{
			"combine MultiErrors",
			[]error{&multiError{[]Error{errFoo, errBar}}, &multiError{[]Error{errBaz}}},
			&multiError{[]Error{errFoo, errBar, errBaz}},
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

func TestFormatError(t *testing.T) {
	for _, tc := range []struct {
		name      string
		multiline bool
		errors    []error
		want      string
	}{
		{
			"build golang errors without new line",
			false,
			[]error{errBarRaw},
			singlelineError(),
		},
		{
			"build golang errors with multi line",
			true,
			[]error{errFooRaw, errBarRaw},
			multilineError(),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var errs MultiError
			for _, err := range tc.errors {
				errs = Append(errs, err)
			}
			err := FormatError(tc.multiline, errs)
			if tc.want != err {
				t.Errorf("FormatError() got: %s; want: %s", err, tc.want)
			}
		})
	}
}
