package dialog

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strconv"

	"github.com/golang/glog"
	"github.com/google/nomos/pkg/process/exec"
	"github.com/pkg/errors"
)

// Menu offers a number of options to the user to choose from.  The user can
// select at most one.
type Menu struct {
	opts Options

	// the name of the dialog command to execute.  "--menu" for the menu.
	subcommand string

	// menuHeight is the height of the menu.
	menuHeight int

	// items is the unpacked list of menu options to display, alternating the
	// menu tag and the corresponding long option text.  Used MenuItem to
	// specify.
	items []string

	// output contains the result of menu selection.
	output io.ReadWriter

	// cmd is the execution context for the underlying dialog program.
	cmd exec.Context
}

// MenuOption is a single setter of an option for the dialog menu.
type MenuOption func(m *Menu)

// MenuHeight sets the height of the menu to the supplied value.
func MenuHeight(v int) MenuOption {
	return func(m *Menu) {
		m.menuHeight = v
	}
}

// MenuItem creates a MenuOption from input in order to generate elements
// of a menu
func MenuItem(tag, text string) MenuOption {
	return func(m *Menu) {
		if tag == "" {
			tag = fmt.Sprintf("<tag:%v>", len(m.items)/2)
		}
		m.items = append(m.items, tag, text)
	}
}

func (m *Menu) apply(opts ...interface{}) {
	for _, opt := range opts {
		glog.V(10).Infof("menu before: %#v", m)
		switch opt.(type) {
		case Options:
			m.opts = opt.(Options)
		case MenuOption:
			opt.(MenuOption)(m)
		case OptionsFunc:
			opt.(OptionsFunc)(&m.opts)
		default:
			panic(fmt.Sprintf("unsupported option: %T", opt))
		}
		glog.V(10).Infof("menu  after: %#v", m)
	}
}

// NewMenu creates a new menu item, using the supplied options.
func NewMenu(opts ...interface{}) *Menu {
	m := &Menu{opts: NewOptions(), items: []string{}}
	m.apply(opts...)
	const menuOpt = "--menu"
	m.subcommand = menuOpt
	glog.V(10).Infof("menu: %#v", *m)
	return m
}

// cmdline constructs the full command line for the menu.
func (m *Menu) cmdline() []string {
	cmdline := append([]string{dialogCmd}, m.opts.rest...)
	cmdline = append(cmdline, m.subcommand,
		m.opts.message, strconv.Itoa(m.opts.height),
		strconv.Itoa(m.opts.width), strconv.Itoa(m.menuHeight))
	cmdline = append(cmdline, m.items...)
	return cmdline
}

// Display displays the current menu configuration on screen, and does not block
// the execution while the user is selecting an option from the menu.
func (m *Menu) Display() {
	m.output = &bytes.Buffer{}
	m.cmd = exec.NewRedirected(os.Stdin, os.Stdout, m.output)
	m.cmd.Start(context.Background(), m.cmdline()...)
}

// Close returns the tag of the selected menu option, empty string on cancel, or error if any.
// An error may occur if there is a configuration error for the dialog (i.e. a bug).
func (m *Menu) Close() (string, error) {
	if err := m.cmd.Wait(); err != nil {
		if err.Error() == "exit status 1" {
			// dialog returns error 1 when user selects cancel
			return "", nil
		}
		return "", errors.Wrapf(err, "while waiting for dialog menu")
	}
	b, err := ioutil.ReadAll(m.output)
	if err != nil {
		return "", errors.Wrapf(err, "while reading menu selection")
	}
	return string(b), nil
}
