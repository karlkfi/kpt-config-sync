/*
Copyright 2018 The Nomos Authors.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package dialog is a go binding for the dialog program.  Dialog program is a
// utility that displays graphical menu elements in a terminal, to provide some
// user-friendliness in installation programs.
package dialog

import (
	"github.com/google/nomos/pkg/process/exec"
)

var (
	dialogCmd = exec.RequireProgram("dialog")
)

const (
	// Default dimensions of a displayed dialog.
	defaultWidth  = 60
	defaultHeight = 20
)

// Options represent the settings common to all dialog types.
type Options struct {
	// width and height are the dimensions of the dialog box to set.
	// If left unset, they are set to reasonable defaults.
	width, height int

	// The message shown in the dialog where it applies.
	message string

	// The rest of the dialog command line, in case there are options that are
	// not directly supported by the bindings.
	rest []string
}

// OptionsFunc is the type of the options setter.
type OptionsFunc func(*Options)

// NewOptions creates a new set of options.
func NewOptions(opts ...OptionsFunc) Options {
	v := Options{width: defaultWidth, height: defaultHeight, rest: nil}
	for _, opt := range opts {
		opt(&v)
	}
	return v
}

// Rest adds parameters to the default options.  Use for any options that are not
// yet supported by explicit methods.
func Rest(params ...string) OptionsFunc {
	return func(o *Options) {
		o.rest = append(o.rest, params...)
	}
}

// Height sets the height of the dialog, in character heights.
func Height(height int) OptionsFunc {
	return func(o *Options) {
		o.height = height
	}
}

// Width sets the width of the dialog, in character widths.
func Width(width int) OptionsFunc {
	return func(o *Options) {
		o.width = width
	}
}

// Title sets the title to be displayed on top of the dialog box.
func Title(title string) OptionsFunc {
	const titleOpt = "--title"
	return func(o *Options) {
		o.rest = append(o.rest, titleOpt, title)
	}
}

// Backtitle sets the text to be displayed in the top-left corner of the background.
func Backtitle(backtitle string) OptionsFunc {
	const backtitleOpt = "--backtitle"
	return func(o *Options) {
		o.rest = append(o.rest, backtitleOpt, backtitle)
	}
}

// Colors enables interpreting embedded "\Z" sequences, to color and format the
// output.
func Colors() OptionsFunc {
	const colorsOpt = "--colors"
	return func(o *Options) {
		o.rest = append(o.rest, colorsOpt)
	}
}

// Message sets the message to be displayed in the dialog.
func Message(text string) OptionsFunc {
	return func(o *Options) {
		o.message = text
	}
}
