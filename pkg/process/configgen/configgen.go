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

// Package configgen contains the library functions for the configuration generator.
package configgen

import (
	"fmt"

	"strings"

	"github.com/blang/semver"
	"github.com/golang/glog"
	"github.com/google/nomos/pkg/installer/config"
	"github.com/google/nomos/pkg/process/dialog"
	"github.com/pkg/errors"
)

const (
	menuTitle = "Configuration generator options"
	// This is the title message shown in the first menu.
	menuMessage = `Select one of the options below to change the installation options.`
)

type Generator struct {
	// defaultCfg is the starting configuration for the load.
	defaultCfg config.Config
	// currentCfg is the current loaded configuration.
	currentCfg config.Config
	dir        string
	out        string
	version    semver.Version

	// The actions at the top level of the menu.
	actions []Action

	// The default configuration options.
	opts dialog.Options
}

// New creates a new Generator, with the supplied version, working directory,
// initial default configuration and output filename.
func New(version semver.Version, workDir string, cfg config.Config, out string) *Generator {
	opts := dialog.NewOptions(
		dialog.Backtitle(fmt.Sprintf("Configuration generator v%v", version)),
		dialog.Title(menuTitle),
		dialog.Width(100),
		dialog.Height(20),
		dialog.Message("hello!"),
		dialog.Colors(),
	)
	g := &Generator{
		defaultCfg: cfg,
		currentCfg: cfg,
		dir:        workDir,
		out:        out,
		version:    version,
		opts:       opts,
	}
	s := NewSave(g.out, &g.currentCfg)
	g.actions = []Action{
		NewUserForm(g.opts, &g.currentCfg.User),
		NewClusters(g.opts, &g.currentCfg.Contexts),
		NewGitForm(g.opts, g.currentCfg.Git),
		NewSSHForm(g.opts, g.currentCfg.SSH),
		NewInstallAction(s, &g.currentCfg, g.dir),
		s,
		&staticAction{name: "Quit", text: "Quit the configuration generator.", quit: true, implemented: true},
	}
	return g
}

// buildMenu creates a menu based on the current content of the actions. Uses
// the global options specified in opts.
func buildMenu(actions []Action, opts dialog.Options, err error) *dialog.Menu {
	o := []interface{}{opts}
	messages := []string{menuMessage}
	if err != nil {
		messages = append(messages, fmt.Sprintf("\\Z1\\Zb%v\\Zn", err))
	}
	o = append(o, dialog.Message(strings.Join(messages, "\n")))
	for _, action := range actions {
		o = append(o, dialog.MenuItem(action.Name(), action.Text()))
	}
	o = append(o, dialog.MenuHeight(len(actions)))
	return dialog.NewMenu(o...)
}

// Run starts the configuration generator.  This call blocks, returning an
// error in the configuration generation process, if any.
func (g *Generator) Run() error {
	var (
		// The "current" error status in the loop.
		err error
		// The "current" menu selection in the loop.
		sel string
	)
	done := false
	for !done {
		m := buildMenu(g.actions, g.opts, err)
		m.Display()
		sel, err = m.Close()
		if err != nil {
			return errors.Wrapf(err, "configgen.Run(): while selecting options")
		}
		done, err = g.runSelection(sel)
		if err != nil {
			// A non-nil error here will get surfaced in the UI.  Log it anyways.
			glog.Warning(errors.Wrapf(err, "after runSelection"))
		}
	}
	return err
}

func (g *Generator) runSelection(tag string) (bool, error) {
	for _, a := range g.actions {
		if a.Name() != tag {
			continue
		}
		return a.Run()
	}
	return true, fmt.Errorf("tag not found: %q", tag)
}

// Action is a single configuration generator action.
type Action interface {
	// Name returns the action's name.
	Name() string

	// Text returns the action's long text.
	Text() string

	// Runs the action.  Returns done == true if the loop should be ended,
	// or false otherwise.  Returns error if any.
	Run() (bool, error)
}

var _ Action = (*staticAction)(nil)

// staticAction is a dummy action that only has static text.
type staticAction struct {
	// The label and text of the menu item involved.
	name, text string
	// If set, selecting the option will quit the menu loop.
	quit bool
	// If unset, the text label will be prepended to show that the option is not
	// implemented.
	implemented bool
}

// Text implements Action.
func (a *staticAction) Text() string {
	text := a.text
	if !a.implemented {
		text = fmt.Sprintf("[NOT IMPLEMENTED] %v", text)
	}
	return text
}

// Name implements Action.
func (a *staticAction) Name() string {
	return a.name
}

// Run implements Action.
func (a *staticAction) Run() (bool, error) {
	return a.quit, nil
}
