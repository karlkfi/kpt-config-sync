package configgen

import (
	"github.com/golang/glog"
	"github.com/google/nomos/pkg/toolkit/exec"
	"github.com/google/nomos/pkg/toolkit/installer"
	"github.com/google/nomos/pkg/toolkit/installer/config"
	"github.com/pkg/errors"
)

var (
	installerCmd = exec.RequireProgram("installer")
)

var _ Action = (*InstallAction)(nil)

// Runner is the interface for running a foreign Action.
type Runner interface {
	// Run is the command to run to make the change.
	Run() (bool, error)
}

// InstallAction saves the current configuration and runs the installer based
// on it.
type InstallAction struct {
	// A runner that saves the file.
	r Runner

	// Points to the current configuration to use.
	cfg *config.Config

	// The current work directory.
	dir string
}

// NewInstallAction creates a new save action item, using the supplied configuration
// and working directory.
func NewInstallAction(r Runner, cfg *config.Config, dir string) *InstallAction {
	return &InstallAction{r, cfg, dir}
}

// Text implements Action.
func (a *InstallAction) Text() string {
	return "Run the installer with current configuration"
}

// Name implements Action.
func (a *InstallAction) Name() string {
	return "Install"
}

// Run implements Action.  This Run runs the installer with the current configuration
// and then executes the installer command with flags corresponding to that
// configuration.
func (a *InstallAction) Run() (bool, error) {
	done, err := a.r.Run()
	if err != nil {
		return done, errors.Wrapf(err, "while saving configuration")
	}
	i := installer.New(*a.cfg, a.dir)
	err = i.Run()
	if err != nil {
		if glog.V(5) {
			glog.Warningf("installer returned error: %v", err)
		}
		return false, errors.Wrapf(err, "while executing installer")
	}
	return false, nil
}
