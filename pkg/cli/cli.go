/*
Copyright 2017 The Kubernetes Authors.

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

import "github.com/google/stolos/pkg/client/meta"

// KubectlPluginFunction is a function that takes a context and an arg list prefix
type KubectlPluginFunction func(ctx *CommandContext, args []string) error

// RegisterKubectlPluginFunction registers a plugin function, call this in init()
// inside your module.
func RegisterKubectlPluginFunction(argPrefix []string, callback KubectlPluginFunction) {
}

// CommandContext is the command context object, this provides all info passed in by
// kubectl and provides a client object that can be used to talk to the k8s server
// that kubectl is presently talking to.
type CommandContext struct {
	GlobalFlags map[string]string
	LocalFlags  map[string]string
	Name        string
	ShortDesc   string
	LongDesc    string
	Example     string
	Command     string
	Namespace   string
	Client      meta.Interface
}

func NewCommandContext() *CommandContext {
	return &CommandContext{
		GlobalFlags: map[string]string{},
		LocalFlags:  map[string]string{},
	}
}

type registryNode struct {
	subcommands map[string]*registryNode
	function    KubectlPluginFunction
}

func newRegistryNode() *registryNode {
	return &registryNode{}
}

var registry = newRegistryNode()

// Invoke will call the funcion callback with the longest matching arg prefix of args
// and pass remaining args to the function callback.
func (c *CommandContext) Invoke(args []string) {

}
