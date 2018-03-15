package configgen

import (
	"fmt"

	"github.com/google/stolos/pkg/toolkit/dialog"
	"github.com/pkg/errors"
)

var _ Action = (*UserForm)(nil)

const (
	usernameMessage = `Please enter the username of the user that will be installing.

The username you enter here (e.g. someuser@example.com) will temporarily become cluster admin for the purposes of installing the system components.  This setting is reversed when the installation process is over.`
)

// UserForm is the view that allows the user to input an username.
type UserForm struct {
	// The configuration view.
	form *dialog.Form
	// The default setting.
	defaultCfg string
	// The model that is changed upon edit.
	currentCfg *string
}

// NewUserForm returns a new form to set the user name.
func NewUserForm(o dialog.Options, currentCfg *string) *UserForm {
	f := &UserForm{defaultCfg: *currentCfg, currentCfg: currentCfg}
	opts := []interface{}{
		o,
		dialog.MenuHeight(4),
		dialog.Message(usernameMessage),
		dialog.FormItem(
			dialog.FormLabel{
				Text: "Username:", Y: 1, X: 1},
			dialog.FormLabel{
				Text: *f.currentCfg, Y: 1, X: 11},
			inputFieldVisibleLength, inputFieldVisibleLength,
			f.currentCfg,
		),
	}
	f.form = dialog.NewForm(opts...)
	return f
}

// Name implements Action.
func (u *UserForm) Name() string {
	return "User"
}

// Text implements Action.
func (u *UserForm) Text() string {
	text := "Set up the user that will act as cluster admin"
	if *u.currentCfg == "" {
		text = fmt.Sprintf("%v [UNSET]", text)
	} else {
		text = fmt.Sprintf("%v [%v]", text, *u.currentCfg)
	}
	return text
}

// Run implements Action.
func (u *UserForm) Run() (bool, error) {
	u.form.Display()
	if err := u.form.Close(); err != nil {
		return false, errors.Wrapf(err, "while filling in git config form")
	}
	return false, nil
}
