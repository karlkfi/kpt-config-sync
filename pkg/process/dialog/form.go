package dialog

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

// varPos remembers the view variable and its index in the form fields.
type varPos struct {
	v     *string
	index int
}

// Form is a dialog type that offers the user the option to supply several
// textual inputs.
type Form struct {
	// The underlying basic menu widget.
	m *Menu

	// The list of variables that will accept the values of the filled form items.
	vars []varPos
}

// Label is a single label for the form.
type Label struct {
	// The text content of the label.  The label may be either editable (in
	// which case it is a static text on the screen), or can be editable (in
	// which case it is an input field).
	Text string

	// X and Y coordinates, relative to the top-left corner of the form, where
	// the label should appear.
	X, Y int
}

// Field describes a single field entry.
type Field struct {
	// The model variable that is changed on successful input.
	Input *string

	// X and Y coordinates, relative to the top-left corner of the form, where
	// the label should appear.
	X, Y int

	// ViewLen is the length of the input field.
	ViewLen int

	// MaxLen is the maximum allowable length of the input.  It may be larger
	// than ViewLen, in which case the field will scroll to show longer input.
	MaxLen int
}

// FormOptionFunc is a function type for options setters.
type FormOptionFunc func(f *Form)

// FormItem declares a new form field.  dest is the destination variable that
// will accept the form's value.  tag is the short label to display for a form,
// and tagX and tagY are its coordinates.  formLen is the length of the display
// field, and maxLen is the maximum length of the string input.  dest is the
// variable to place the form result into.
func FormItem(tag Label, item Field) FormOptionFunc {
	return func(f *Form) {
		if tag.Text == "" {
			tag.Text = fmt.Sprintf("<tag:%v>", len(f.m.items)/8)
		}
		f.m.items = append(f.m.items,
			tag.Text, strconv.Itoa(tag.Y), strconv.Itoa(tag.X),
			*item.Input)
		// Remember the index of the view variable.
		f.vars = append(f.vars, varPos{item.Input, len(f.m.items) - 1})

		f.m.items = append(f.m.items, strconv.Itoa(item.Y), strconv.Itoa(item.X),
			strconv.Itoa(item.ViewLen), strconv.Itoa(item.MaxLen))
	}
}

// NewForm creates a new input form with the given options.
func NewForm(opts ...interface{}) *Form {
	c := Form{m: NewMenu(), vars: []varPos{}}
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
	// Reevaluate the arguments to display.
	for _, vp := range f.vars {
		f.m.items[vp.index] = *vp.v
	}
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
	// Pair up the selection outputs and the variables that should receive
	// these outputs.
	j := 0
	for i, s := range sel {
		if i >= len(f.vars) {
			break
		}
		// Pair up the selection with the variables.
		*f.vars[j].v = s
		j++
	}
	return nil
}
