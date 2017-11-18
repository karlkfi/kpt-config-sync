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

package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/golang/glog"
	"github.com/google/stolos/pkg/cli"
)

const (
	globalFlagPrefix        = "KUBECTL_PLUGINS_GLOBAL_FLAG_"
	localFlagPrefix         = "KUBECTL_PLUGINS_LOCAL_FLAG_"
	kubectlDescriptorPrefix = "KUBECTL_PLUGINS_DESCRIPTOR_"
)

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

func main() {
	flag.Parse()
	args := flag.Args()

	context := cli.NewCommandContext()

	flag.Lookup("logtostderr").Value.Set("true")
	flag.Lookup("v").Value.Set("1")

	glog.V(1).Infof("Plugin invoked with %d args: %s\n", len(args), strings.Join(args, " "))
	glog.V(1).Infof("Plugin invoked with env state:\n")
	// Setup flags from environment variables.
	for _, pair := range os.Environ() {
		kvList := strings.Split(pair, "=")
		key := kvList[0]
		value := kvList[1]

		switch {
		case strings.HasPrefix(key, globalFlagPrefix):
			flagName := strings.ToLower(key[len(globalFlagPrefix):])
			glog.V(1).Infof("  ENV: %s=%s", key, value)
			glog.V(1).Infof("  kubectl flag: --%s=%s\n", flagName, value)
			setFlag(flagName, value)
			context.GlobalFlags[flagName] = value

		case strings.HasPrefix(key, localFlagPrefix):
			flagName := strings.ToLower(key[len(localFlagPrefix):])
			glog.V(1).Infof("  ENV: %s=%s", key, value)
			glog.V(1).Infof("  Local flag: --%s=%s\n", flagName, value)
			setFlag(flagName, value)
			context.LocalFlags[flagName] = value

		case strings.HasPrefix(key, kubectlDescriptorPrefix):
			descriptor := strings.ToLower(key[len(kubectlDescriptorPrefix):])
			glog.V(1).Infof("  Descriptor %s: %s\n", descriptor, value)
			switch descriptor {
			case "name":
				context.Name = value
			case "short_desc":
				context.ShortDesc = value
			case "long_desc":
				context.LongDesc = value
			case "example":
				context.Example = value
			case "command":
				context.Command = value
			}

		case key == "KUBECTL_PLUGINS_CURRENT_NAMESPACE":
			glog.V(1).Infof("  Plugin namespace: %s\n", value)
			context.Namespace = value
		}
	}

	if len(args) == 0 {
		fmt.Printf("No command specified.")
		os.Exit(1)
	}

}
