package dialog

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"
)

// Checklist offers a number of options to the user to choose from.  The user
// can select zero or more.
type Checklist struct {
	m *Menu
}

// ChecklistItem creates a new item in a checklist, with a short tag, a longer
// explanation text and an indicator whether it should be selected initially
// or not.
func ChecklistItem(tag, text string, selected bool) MenuOption {
	return func(m *Menu) {
		if tag == "" {
			tag = fmt.Sprintf("<tag:%v>", len(m.items)/3)
		}
		sel := "off"
		if selected {
			sel = "on"
		}
		m.items = append(m.items, tag, text, sel)
	}
}

// NewChecklist creates a new checklist.
func NewChecklist(opts ...interface{}) *Checklist {
	c := Checklist{m: NewMenu()}
	const checklistOpt = "--checklist"
	c.m.subcommand = checklistOpt
	// For now all menu options are also checklist options.
	c.m.apply(opts...)
	return &c
}

// Display displays the checklist dialog in a non-blocking way.  Call Close()
// to collect the output.
func (c *Checklist) Display() {
	c.m.Display()
}

// Close closes the checklist dialog and returns the tags that have been selected.
// Close may be called only once in the lifetime of a checklist.
func (c *Checklist) Close() ([]string, error) {
	sels, err := c.m.Close()
	if err != nil {
		return nil, errors.Wrapf(err, "Checklist:Close(): while closing the underlying menu")
	}
	sel := strings.Split(sels, " ")
	return sel, nil
}
