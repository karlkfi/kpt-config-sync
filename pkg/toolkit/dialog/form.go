package dialog

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

// Form is a dialog type that offers the user the option to supply several
// textual inputs.
type Form struct {
	// The underlying basic menu widget.
	m *Menu

	// The list of variables that will accept the values of the filled form items.
	vars []*string
}

// FormLabel is a single label for the form.
type FormLabel struct {
	// The text content of the label.  The label may be either editable (in
	// which case it is a static text on the screen), or can be editable (in
	// which case it is an input field).
	Text string

	// X and Y coordinates, relative to the top-left corner of the form, where
	// the label should appear.
	X, Y int
}

// FormOptionFunc is a function type for options setters.
type FormOptionFunc func(f *Form)

// FormItem declares a new form field.  dest is the destination variable that
// will accept the form's value.  tag is the short label to display for a form,
// and tagX and tagY are its coordinates.  formLen is the length of the display
// field, and maxLen is the maximum length of the string input.  dest is the
// variable to place the form result into.
func FormItem(tag, item FormLabel, formLen, maxLen int, dest *string) FormOptionFunc {
	return func(f *Form) {
		if tag.Text == "" {
			tag.Text = fmt.Sprintf("<tag:%v>", len(f.m.items)/8)
		}
		f.m.items = append(f.m.items,
			tag.Text, strconv.Itoa(tag.Y), strconv.Itoa(tag.X),
			item.Text, strconv.Itoa(item.Y), strconv.Itoa(item.X),
			strconv.Itoa(formLen), strconv.Itoa(maxLen))
		f.vars = append(f.vars, dest)
	}
}

// NewForm creates a new input form with the given options.
func NewForm(opts ...interface{}) *Form {
	c := Form{m: NewMenu(), vars: []*string{}}
	const formOpt = "--form"
	c.m.subcommand = formOpt
	c.apply(opts...)
	return &c
}

func (f *Form) apply(opts ...interface{}) {
	var rest []interface{}
	for _, opt := range opts {
		switch opt.(type) {
		case FormOptionFunc:
			opt.(FormOptionFunc)(f)
		default:
			rest = append(rest, opt)
		}
	}
	f.m.apply(rest...)
}

// Display displays the form dialog in a non-blocking way.  Call Close()
// to collect the output.
func (f *Form) Display() {
	f.m.Display()
}

// Close returns the error (if any) from displaying the form.  It may be called
// at most once in the lifetime of the form, and only after Display() has been
// called.
func (f *Form) Close() error {
	sels, err := f.m.Close()
	if err != nil {
		return errors.Wrapf(err, "Form::Close(): while closing form")
	}
	sel := strings.Split(sels, "\n")
	j := 0
	for _, s := range sel {
		if s == "" {
			continue
		}
		// Pair up the selection with the variables.
		*f.vars[j] = s
		j++
	}
	return nil
}
