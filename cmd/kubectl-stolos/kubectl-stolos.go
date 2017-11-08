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
	"fmt"
	"os"
	"strings"
)

const (
	globalFlagPrefix = "KUBECTL_PLUGINS_GLOBAL_FLAG_"
	localFlagPrefix  = "KUBECTL_PLUGINS_LOCAL_FLAG_"
)

func main() {
	args := os.Args[1:]

	fmt.Printf("Plugin invoked with %d args: %s\n", len(args), strings.Join(args, " "))
	fmt.Printf("Plugin invoked with env state:\n")
	for _, pair := range os.Environ() {
		kvList := strings.Split(pair, "=")
		key := kvList[0]
		value := kvList[1]

		switch {
		case strings.HasPrefix(key, globalFlagPrefix):
			flagName := key[len(globalFlagPrefix):]
			fmt.Printf(" kubectl flag: --%s=%s\n", flagName, value)
		case strings.HasPrefix(key, localFlagPrefix):
			flagName := key[len(localFlagPrefix):]
			fmt.Printf(" Local flag: --%s=%s\n", flagName, value)
		case strings.HasPrefix(key, "KUBECTL_PLUGINS_DESCRIPTOR_"):
			fmt.Printf(" Descriptor: %s\n", value)
		case key == "KUBECTL_PLUGINS_CURRENT_NAMESPACE":
			fmt.Printf(" Plugin namespace: %s\n", value)
		}
	}
}
