package configgen

import (
	"fmt"
	"strconv"

	"github.com/google/go-cmp/cmp"
	"github.com/google/stolos/pkg/toolkit/dialog"
	"github.com/google/stolos/pkg/toolkit/installer/config"
	"github.com/pkg/errors"
)

const (
	gitSettingsMessage = `Please set the git repository parameters.

These parameters are used by the policy importer to contact the git repository
acting as the source of truth for the policies.`

	// The default visible length of each input field.
	inputFieldVisibleLength = 60

	// The default maximum input for each field in this form.
	inputFieldMaxLength = 200

	// The column at which the form text input starts.
	gitFormTextColumn = 2

	// The column at which the input field text starts.
	gitFormInputColumn = 43

	// Rows to display various input fields in.
	gitRepoRow       = 2
	branchToSyncRow  = 4
	rootPolicyDirRow = 6
	syncWaitRow      = 8
)

var _ Action = (*GitForm)(nil)

// GitForm is the view that controls the git configuration.
type GitForm struct {
	// The configuration view.
	form *dialog.Form
	// The default settings.
	defaultCfg config.GitConfig
	// The model to modify when editing a new form.
	currentConfig *config.GitConfig
	// The sync wait period in seconds, represented as string.
	syncWaitAsString string
}

// NewGitForm returns a new form for querying git options.
func NewGitForm(o dialog.Options, cfg *config.GitConfig) *GitForm {
	gf := &GitForm{defaultCfg: *cfg, currentConfig: cfg}

	const (
		gitSyncRepoText   = "Git repository (GIT_SYNC_REPO):"
		branchToSyncText  = "Branch to sync (GIT_SYNC_BRANCH):"
		rootPolicyDirText = "Root policy directory (ROOT_POLICY_DIR):"
		syncWaitText      = "Sync wait (in seconds) (GIT_SYNC_WAIT):"
	)

	gf.syncWaitAsString = fmt.Sprintf("%v", cfg.SyncWaitSeconds)
	opts := []interface{}{
		o,
		dialog.MenuHeight(10),
		dialog.Message(gitSettingsMessage),
		dialog.FormItem(
			dialog.Label{
				Text: gitSyncRepoText, Y: gitRepoRow, X: gitFormTextColumn},
			dialog.Field{
				Input: &gf.currentConfig.SyncRepo, Y: gitRepoRow, X: gitFormInputColumn,
				ViewLen: inputFieldVisibleLength, MaxLen: inputFieldVisibleLength},
		),
		dialog.FormItem(
			dialog.Label{
				Text: branchToSyncText, Y: branchToSyncRow, X: gitFormTextColumn},
			dialog.Field{
				Input: &gf.currentConfig.SyncBranch, Y: branchToSyncRow, X: gitFormInputColumn,
				ViewLen: inputFieldVisibleLength, MaxLen: inputFieldVisibleLength},
		),
		dialog.FormItem(
			dialog.Label{
				Text: rootPolicyDirText, Y: rootPolicyDirRow, X: gitFormTextColumn},
			dialog.Field{
				Input: &gf.currentConfig.RootPolicyDir, Y: rootPolicyDirRow, X: gitFormInputColumn,
				ViewLen: inputFieldVisibleLength, MaxLen: inputFieldVisibleLength},
		),
		dialog.FormItem(
			dialog.Label{
				Text: syncWaitText, Y: syncWaitRow, X: gitFormTextColumn},
			dialog.Field{
				Input: &gf.syncWaitAsString,
				Y:     syncWaitRow, X: gitFormInputColumn,
				ViewLen: 8, MaxLen: 8},
		),
	}
	gf.form = dialog.NewForm(opts...)
	return gf
}

// Name implements Actions.
func (g *GitForm) Name() string {
	return "Git"
}

// Text implements Actions.
func (g *GitForm) Text() string {
	text := "Configure the Git Policy Importer module"
	if cmp.Equal(g.defaultCfg, *g.currentConfig) {
		text = fmt.Sprintf("%v [DEFAULT]", text)
	}
	if g.currentConfig.SyncRepo == "" {
		text = fmt.Sprintf("%v [UNSET]", text)
	}
	return text
}

// Run implements Actions.
func (g *GitForm) Run() (bool, error) {
	g.form.Display()
	if err := g.form.Close(); err != nil {
		return false, errors.Wrapf(err, "while filling in git config form")
	}
	c, err := strconv.Atoi(g.syncWaitAsString)
	if err != nil {
		g.currentConfig.SyncWaitSeconds = g.defaultCfg.SyncWaitSeconds
	}
	g.currentConfig.SyncWaitSeconds = int64(c)
	return false, nil
}
