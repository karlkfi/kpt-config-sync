/*
Copyright 2017 The Nomos Authors.

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

package cli

import (
	"strings"

	"github.com/pkg/errors"
)

// KubectlPluginFunction is a function that implements a kubectl binary plugin extension command.
// It takes a populated context and an arg list of the remaining args that were passed to the command.
type KubectlPluginFunction func(ctx *CommandContext, args []string) error

// registryNode implements a trie-like structure where each level is indexed by string rather than
// character.  The value attached to the structure is a function which is used for dispatching
// the CLI call.
type registryNode struct {
	subcommands map[string]*registryNode // Subnodes keyed by arg name
	function    KubectlPluginFunction    // The function callback for this particular node.
}

func newRegistryNode() *registryNode {
	return &registryNode{
		subcommands: map[string]*registryNode{},
	}
}

// getNode returns the node which exactly matches the arg list.
func (r *registryNode) getNode(args []string) *registryNode {
	if len(args) == 0 {
		return r
	}

	cmd := args[0]
	rest := args[1:]
	subNode := r.subcommands[cmd]
	if subNode == nil {
		return nil
	}
	return subNode.getNode(rest)
}

// getNodeByPrefix returns the registryNode which matches the longest arg list specified.  Extra args
// that were not matched will be returned in the second return value of the function.
func (r *registryNode) getNodeByPrefix(args []string) (*registryNode, []string) {
	if len(args) == 0 {
		return r, []string{}
	}

	cmd := args[0]
	rest := args[1:]
	subNode := r.subcommands[cmd]
	if subNode == nil {
		return r, args
	}
	return subNode.getNodeByPrefix(rest)
}

// getOrCreateNode returns a node which exactly matches the arg list.  The node and all parent nodes
// will be created if they do not exist.
func (r *registryNode) getOrCreateNode(args []string) *registryNode {
	if len(args) == 0 {
		return r
	}

	cmd := args[0]
	rest := args[1:]
	subNode := r.subcommands[cmd]
	if subNode == nil {
		subNode = newRegistryNode()
		r.subcommands[cmd] = subNode
	}
	return subNode.getOrCreateNode(rest)
}

func (r *registryNode) addCommand(args []string, callback KubectlPluginFunction) error {
	cmdNode := r.getOrCreateNode(args)
	if cmdNode.function != nil {
		return errors.Errorf("Cannot create node %q", strings.Join(args, " "))
	}
	cmdNode.function = callback
	return nil
}

var registry = newRegistryNode()

// RegisterKubectlPluginFunction registers a plugin function, call this in init inside your package
// or use cli/registration/registration.go
func RegisterKubectlPluginFunction(argPrefix []string, callback KubectlPluginFunction) {
	err := registry.addCommand(argPrefix, callback)
	if err != nil {
		panic(err)
	}
}
