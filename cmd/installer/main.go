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

// Package configgen contains the utility for generating configurations.
package main

import (
	"flag"
	"os"
	"path"
	"regexp"

	"github.com/blang/semver"
	"github.com/golang/glog"
	"github.com/google/nomos/pkg/installer"
	"github.com/google/nomos/pkg/installer/config"
	"github.com/pkg/errors"
)

var (
	workDir       = flag.String("work_dir", "", "The working directory for the configgen.  If not set, defaults to the directory where the configgen is run.")
	suggestedUser = flag.String("suggested_user", "", "The user to run the installation as.")
	configFile    = flag.String("config", "", "The file name containing the installer configuration.")
	uninstall     = flag.String("uninstall", "", "If set, the supplied clusters will be uninstalled.")
	useCurrent    = flag.Bool("use_current_context", false, "If set, and if the list of clusters in the install config is empty, use current context to install into.")
)

// version parses vstr, which could be of the form "prefix1.2.3-blah+blah".
func versionOrDie(vstr string) semver.Version {
	// Rewind the regexp to the first digit so it can be parsed as a semver.
	var digitRe = regexp.MustCompile("[0-9]")
	i := digitRe.FindStringIndex(vstr)
	if len(i) == 0 || i[0] < 0 {
		glog.Exit(errors.Errorf("while parsing --version=%v", vstr))
	}
	v, err := semver.Parse(vstr[i[0]:])
	if err != nil {
		glog.Exit(errors.Wrapf(err, "while parsing --version=%v", vstr))
	}
	return v
}

func main() {
	flag.Parse()

	if *configFile == "" {
		glog.Exit("--config is required in batch mode")
		flag.Usage()
		os.Exit(1)
	}
	glog.V(10).Infof("Starting installer.")

	file, err := os.Open(*configFile)
	if err != nil {
		glog.Exit(errors.Wrapf(err, "while opening: %q", *configFile))
	}

	config, err := config.Load(file)
	if err != nil {
		glog.Exit(errors.Wrapf(err, "while loading: %q", *configFile))
	}
	glog.V(1).Infof("Using config: %#v", config)
	if config.User == "" && *suggestedUser != "" {
		// If the configuration has no user specified, but the user is suggested
		// instead, use that user then.
		config.User = *suggestedUser
	}

	dir := path.Dir(os.Args[0])
	if *workDir != "" {
		dir = *workDir
	}
	i := installer.New(config.ExpandVarsCopy(), dir)

	if *uninstall != "" {
		if err := i.Uninstall(*uninstall, *useCurrent); err != nil {
			glog.Exit(errors.Wrapf(err, "uninstallation failed"))
		}
		glog.Infof("Uninstall successful!")
		return
	}
	if err := i.Run(*useCurrent); err != nil {
		glog.Exit(errors.Wrapf(err, "installation failed"))
	}
	glog.Infof("Install successful!")
}
