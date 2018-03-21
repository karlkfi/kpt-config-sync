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

// Package cli handles setting up and dispatching commands to a kubernetes binary plugin. Many
// of the terms here refer to terms that are defined by the kubectl plugin interface.
// https://kubernetes.io/docs/tasks/extend-kubectl/kubectl-plugins/
package cli

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/google/nomos/pkg/client/restconfig"

	"github.com/golang/glog"
	"github.com/google/nomos/pkg/client/meta"
	"github.com/pkg/errors"
)

// Defines prefixes for environment variables that kubectl passes to the binary plugin
const (
	globalFlagPrefix        = "KUBECTL_PLUGINS_GLOBAL_FLAG_"
	localFlagPrefix         = "KUBECTL_PLUGINS_LOCAL_FLAG_"
	kubectlDescriptorPrefix = "KUBECTL_PLUGINS_DESCRIPTOR_"
)

// CommandContext is the command context object, this provides all info passed in by
// kubectl and provides a client object that can be used to talk to the k8s server
// that kubectl is presently talking to.  This will be provided to KubectlPluginFunction registered
// for commands with RegisterKubectlPluginFunction.  See the official documentation for more info:
// https://kubernetes.io/docs/tasks/extend-kubectl/kubectl-plugins/
type CommandContext struct {
	// These fields are passed via environment variables from the kubectl plugin interface.
	GlobalFlags map[string]string // All "global" kubectl flags
	LocalFlags  map[string]string // All "local" command specific flags
	Name        string            // name of command
	ShortDesc   string            // short description
	LongDesc    string            // long description
	Example     string            // example text
	Command     string            // The command as given on the CLI
	// The namespace to operate on if the command is namespace-specific, otherwise this can be ignored
	Namespace string

	// Flags that affect how the client connects
	asUser string // From the --as flag
	// TODO: add support for --as_group

	// The client interface, this will always be set on invoking the registered callback
	Client meta.Interface
}

func newCommandContext() *CommandContext {
	return &CommandContext{
		GlobalFlags: map[string]string{},
		LocalFlags:  map[string]string{},
	}
}

// NewCommandContext creates a new command context
func NewCommandContext() (*CommandContext, error) {
	ctx := newCommandContext()
	ctx.initEnvironment()
	if err := ctx.setupClient(); err != nil {
		return nil, err
	}
	return ctx, nil
}

func setFlag(name, value string) {
	if name == "log_backtrace_at" && value == ":0" {
		// This is a bug in glog, it expects "", but generates ":0" for the empty value
		return
	}

	flagObj := flag.Lookup(name)
	if flagObj != nil {
		err := flagObj.Value.Set(value)
		if err != nil {
			glog.Errorf("Failed to set flag \"--%s=%s\": %s", name, value, err)
		}
	} else {
		glog.V(1).Infof(" Flag --%s not defined\n", name)
	}
}

// Invoke will call the funcion callback with the longest matching arg prefix of args
// and pass remaining args to the function callback.
func (c *CommandContext) Invoke(args []string) error {
	cmdNode, remainingArgs := registry.getNodeByPrefix(args)
	switch {
	case cmdNode == registry:
		return errors.Errorf("No command specified")
	case cmdNode.function == nil:
		return errors.Errorf("No such command %q", strings.Join(args, " "))
	}
	return cmdNode.function(c, remainingArgs)
}

// InitEnvironment handles setting flags and other values from the environment variables that kubectl
// sets when invoking the binary plugin.
func (c *CommandContext) initEnvironment() {
	// Set logging related flags prior to doing anything that might log.
	for _, flagName := range []string{"v", "logtostderr"} {
		envName := fmt.Sprintf("%s%s", globalFlagPrefix, strings.ToUpper(flagName))
		if val, ok := os.LookupEnv(envName); ok {
			flag.Lookup(flagName).Value.Set(val)
		}
	}

	glog.V(5).Infof("Plugin invoked with env state:\n")
	// Setup flags from environment variables.
	for _, pair := range os.Environ() {
		kvList := strings.SplitN(pair, "=", 2)
		key := kvList[0]
		value := kvList[1]

		switch {
		case strings.HasPrefix(key, globalFlagPrefix):
			flagName := strings.ToLower(key[len(globalFlagPrefix):])
			glog.V(5).Infof("  ENV: %s=%s", key, value)
			glog.V(5).Infof("  kubectl flag: --%s=%s\n", flagName, value)
			switch flagName {
			case "as":
				c.asUser = value
			default:
				setFlag(flagName, value)
				c.GlobalFlags[flagName] = value
			}

		case strings.HasPrefix(key, localFlagPrefix):
			flagName := strings.ToLower(key[len(localFlagPrefix):])
			glog.V(5).Infof("  ENV: %s=%s", key, value)
			glog.V(5).Infof("  Local flag: --%s=%s\n", flagName, value)
			setFlag(flagName, value)
			c.LocalFlags[flagName] = value

		case strings.HasPrefix(key, kubectlDescriptorPrefix):
			descriptor := strings.ToLower(key[len(kubectlDescriptorPrefix):])
			glog.V(5).Infof("  Descriptor %s: %s\n", descriptor, value)
			switch descriptor {
			case "name":
				c.Name = value
			case "short_desc":
				c.ShortDesc = value
			case "long_desc":
				c.LongDesc = value
			case "example":
				c.Example = value
			case "command":
				c.Command = value
			}

		case key == "KUBECTL_PLUGINS_CURRENT_NAMESPACE":
			glog.V(5).Infof("  Plugin namespace: %s\n", value)
			c.Namespace = value
		}
	}

	glog.V(5).Infof("GlobalFlags: %#v", c.GlobalFlags)
	glog.V(5).Infof("LocalFlags: %#v", c.LocalFlags)
}

// setupClient will set up the client for connecting to the kubernetes host.
func (c *CommandContext) setupClient() error {
	restConfig, err := restconfig.NewRestConfig()
	if err != nil {
		return err
	}

	if c.asUser != "" {
		restConfig.Impersonate.UserName = c.asUser
	}

	client, err := meta.NewForConfig(restConfig)
	if err != nil {
		return err
	}
	c.Client = client
	return nil
}
