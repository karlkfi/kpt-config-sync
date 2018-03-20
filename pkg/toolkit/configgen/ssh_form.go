package configgen

import (
	"fmt"

	"github.com/google/go-cmp/cmp"
	"github.com/google/nomos/pkg/toolkit/dialog"
	"github.com/google/nomos/pkg/toolkit/installer/config"
	"github.com/pkg/errors"
)

const (
	sshSettingsMessage = `Please enter the SSH settings.

SSH settings are used to access git repositories through the SSH protocol, e.g.
in case of public git repositories.  SSH offers a fairly robust way to access
such repositories.`

	// The column at which the form text input starts.
	sshFormTextColumn = 2

	// The column at which the input field text starts.
	sshFormInputColumn = 38

	// The default visible length of each input field.
	sshInputFieldVisibleLength = 60

	// The default maximum input for each field in this form.
	sshInputFieldMaxLength = 200

	// Rows to display various input fields in.
	privateKeyFilenameRow = 2
	knownHostsFilenameRow = 4
)

var _ Action = (*SSHForm)(nil)

type SSHForm struct {
	// The configuration view.
	form *dialog.Form
	// The default settings.
	defaultCfg config.SshConfig
	// The model to modify when editing a new form.
	currentConfig *config.SshConfig
}

// NewSSHForm returns a new form for querying SSH options.
func NewSSHForm(o dialog.Options, cfg *config.SshConfig) *SSHForm {
	sf := &SSHForm{defaultCfg: *cfg, currentConfig: cfg}

	const (
		privateKeyFilenameText = "Private key filename:"
		knownHostsFilenameText = "Known hosts filename (known_hosts)"
	)
	opts := []interface{}{
		o,
		dialog.MenuHeight(8),
		dialog.Message(sshSettingsMessage),
		dialog.FormItem(
			dialog.Label{
				Text: privateKeyFilenameText,
				Y:    privateKeyFilenameRow,
				X:    sshFormTextColumn,
			},
			dialog.Field{
				Input:   &sf.currentConfig.PrivateKeyFilename,
				Y:       privateKeyFilenameRow,
				X:       sshFormInputColumn,
				ViewLen: sshInputFieldVisibleLength,
				MaxLen:  sshInputFieldMaxLength,
			},
		),
		dialog.FormItem(
			dialog.Label{
				Text: knownHostsFilenameText,
				Y:    knownHostsFilenameRow,
				X:    sshFormTextColumn,
			},
			dialog.Field{
				Input:   &sf.currentConfig.KnownHostsFilename,
				Y:       knownHostsFilenameRow,
				X:       sshFormInputColumn,
				ViewLen: sshInputFieldVisibleLength,
				MaxLen:  sshInputFieldMaxLength,
			},
		),
	}
	sf.form = dialog.NewForm(opts...)
	return sf
}

// Name implements Action.
func (s *SSHForm) Name() string {
	return "SSH"
}

// Text implements Action.
func (s *SSHForm) Text() string {
	text := "Configure SSH access for the Git Policy Importer"
	if cmp.Equal(s.defaultCfg, *s.currentConfig) {
		text = fmt.Sprintf("%v [DEFAULT]", text)
	}
	if s.currentConfig.PrivateKeyFilename == "" {
		text = fmt.Sprintf("%v [UNSET]", text)
	}
	return text
}

// Run implements Action.
func (s *SSHForm) Run() (bool, error) {
	s.form.Display()
	if err := s.form.Close(); err != nil {
		return false, errors.Wrapf(err, "while filling in git config form")
	}
	return false, nil
}
