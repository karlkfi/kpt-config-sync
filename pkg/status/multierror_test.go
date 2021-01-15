package status

import (
	"fmt"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/pkg/errors"
)

var errFoo = UndocumentedError("foo")
var errBar = APIServerError(errors.New("bar"), "qux")
var errBaz = UndocumentedError("baz")

var errFooRaw = errors.New("raw foo")
var errBarRaw = errors.New("raw bar")

func multiLineError() string {
	b := strings.Builder{}
	b.WriteString(fmt.Sprintf("%s\n\n\n", "2 error(s)"))
	b.WriteString(fmt.Sprintf("%s\n\n", "[1] KNV9999: raw bar"))
	b.WriteString(fmt.Sprintf("%s\n\n\n", urlBase+"9999"))
	b.WriteString(fmt.Sprintf("%s\n\n", "[2] KNV9999: raw foo"))
	b.WriteString(fmt.Sprintf("%s\n", urlBase+"9999"))
	return b.String()
}

func singleLineError() string {
	b := strings.Builder{}
	b.WriteString(fmt.Sprintf("%s\n", "1 error(s) "))
	b.WriteString("[1] KNV9999: raw bar  ")
	b.WriteString(urlBase + "9999 ")
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

			switch {
			case tc.want == nil && errs == nil:
				// Nothing to check; successful test.
			case tc.want == nil && errs != nil:
				t.Errorf("got %d errors; want 0 errors", len(errs.Errors()))
				t.Errorf("got %v; want nil", errs)
			case tc.want != nil && errs == nil:
				t.Errorf("got nil; want %v", tc.want)
			case tc.want != nil && errs != nil:
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
			singleLineError(),
		},
		{
			"build golang errors with multi line",
			true,
			[]error{errFooRaw, errBarRaw},
			multiLineError(),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var errs MultiError
			for _, err := range tc.errors {
				errs = Append(errs, err)
			}
			got := FormatError(tc.multiline, errs)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Error(diff)
			}
		})
	}
}

func TestErrors(t *testing.T) {
	var nilMultiError multiError
	for _, tc := range []struct {
		name   string
		errors multiError
	}{
		{"a nil multiError has no errors", nilMultiError},
		{"an empty multiError has no errors", multiError{errs: []Error{}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			errs := tc.errors.Errors()
			if errs != nil {
				t.Errorf("multiError.Errors() = %v, want nil", errs)
			}
		})
	}
}
