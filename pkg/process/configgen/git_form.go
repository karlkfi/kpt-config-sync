package configgen

import (
	"fmt"
	"strconv"

	"strings"

	"github.com/google/go-cmp/cmp"
	"github.com/google/nomos/pkg/installer/config"
	"github.com/google/nomos/pkg/process/dialog"
	"github.com/pkg/errors"
)

const (
	gitSettingsMessage = `Please set the git repository parameters.

These parameters are used by the policy importer to contact the git repository
acting as the source of truth for the policies.`

	// The default visible length of each input field.
	inputFieldVisibleLength = 60

	// The column at which the form text input starts.
	gitFormTextColumn = 2

	// The column at which the input field text starts.
	gitFormInputColumn = 43

	// Rows to display various input fields in.
	gitRepoRow       = 2
	useSshRow        = 4
	branchToSyncRow  = 6
	rootPolicyDirRow = 8
	syncWaitRow      = 10
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
	// Toggling ssh sync, represented as string (Y/n).
	useSshAsString string
	// The sync wait period in seconds, represented as string.
	syncWaitAsString string
}

// NewGitForm returns a new form for querying git options.
func NewGitForm(o dialog.Options, cfg *config.GitConfig) *GitForm {
	gf := &GitForm{defaultCfg: *cfg, currentConfig: cfg}

	const (
		gitSyncRepoText   = "Git repository (GIT_SYNC_REPO):"
		useSshText        = "Sync repo using ssh (Y/n) (GIT_SYNC_SSH):"
		branchToSyncText  = "Branch to sync (GIT_SYNC_BRANCH):"
		rootPolicyDirText = "Root policy directory (POLICY_DIR):"
		syncWaitText      = "Sync wait (in seconds) (GIT_SYNC_WAIT):"
	)

	gf.syncWaitAsString = fmt.Sprintf("%v", cfg.SyncWaitSeconds)
	gf.useSshAsString = "n"
	if cfg.UseSSH {
		gf.useSshAsString = "Y"
	}
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
				Text: useSshText, Y: useSshRow, X: gitFormTextColumn},
			dialog.Field{
				Input: &gf.useSshAsString,
				Y:     useSshRow, X: gitFormInputColumn,
				ViewLen: 8, MaxLen: 8},
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
	if cmp.Equal(g.defaultCfg, g.currentConfig) {
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
	useSsh := strings.ToLower(g.useSshAsString)
	switch useSsh {
	case "y":
		g.currentConfig.UseSSH = true
	case "n":
		g.currentConfig.UseSSH = false
	default:
		return false, errors.Errorf("while filling in git config form must specify Y or n for GIT_SYNC_SSH field")
	}
	c, err := strconv.Atoi(g.syncWaitAsString)
	if err != nil {
		g.currentConfig.SyncWaitSeconds = g.defaultCfg.SyncWaitSeconds
	}
	g.currentConfig.SyncWaitSeconds = int64(c)
	return false, nil
}
