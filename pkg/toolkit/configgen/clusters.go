/*
Copyright 2018 The Stolos Authors.
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

package configgen

import (
	"fmt"

	"github.com/google/go-cmp/cmp"
	"github.com/google/stolos/pkg/toolkit/dialog"
	"github.com/google/stolos/pkg/toolkit/kubectl"
	"github.com/pkg/errors"
)

const (
	clustersText = "Please select which clusters to install to"

	clustersMenuText = `Make a selection of clusters to install to.

Only clusters that are available to you locally via your kubectl configuration
file are available for selection.  Clusters from the currently active configuration
are checked by default.

Use spacebar to select or unselect, arrow keys to move between entries, Enter
key to confirm, or Tab key to move to the "OK/Cancel" controls.
`
)

var _ Action = (*Clusters)(nil)

// Clusters implements an Action that allows the user to select clusters from
// a list.
type Clusters struct {
	opts dialog.Options

	// The default clusters.
	defaultCfg []string

	// The current selection.  Updated on every successful Run() call.
	currentCfg *[]string
}

// NewClusters creates a new Clusters action.
func NewClusters(
	opts dialog.Options, defaultConfig []string, config *[]string) *Clusters {
	return &Clusters{opts, defaultConfig, config}
}

// Name implements Action.
func (c *Clusters) Name() string {
	return "Clusters"
}

// Text implements Action.
func (c *Clusters) Text() string {
	text := clustersText
	if cmp.Equal(c.defaultCfg, c.currentCfg) {
		text = fmt.Sprintf("%v [DEFAULT]", text)
	}
	clusters := fmt.Sprintf("%v selected", len(*c.currentCfg))
	return fmt.Sprintf("%v [%v]", text, clusters)
}

// Run implements Action.
func (c *Clusters) Run() (bool, error) {
	m := map[string]bool{}
	for _, i := range *c.currentCfg {
		m[i] = true
	}
	cl, err := kubectl.LocalClusters()
	if err != nil {
		return false, errors.Wrapf(err, "while getting local clusters")
	}

	var o []interface{}
	o = append(o, c.opts)
	o = append(o, dialog.Message(clustersMenuText))
	for c, n := range cl.Clusters {
		o = append(o, dialog.ChecklistItem(c, n, m[c]))
	}

	ch := dialog.NewChecklist(o...)
	ch.Display()
	sel, err := ch.Close()
	if err != nil {
		return true, errors.Wrapf(err, "while selecting clusters to install")
	}

	// Apply the setting to the current configuration.
	*c.currentCfg = sel
	return false, nil
}
