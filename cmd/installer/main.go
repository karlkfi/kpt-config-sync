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
	"github.com/google/nomos/pkg/process/configgen"
	"github.com/pkg/errors"
)

var (
	interactive   = flag.Bool("interactive", false, "If set, use the interactive menu driven installer")
	configIn      = flag.String("config_in", "", "The default configuration file to load.")
	configOut     = flag.String("config_out", "generated_config.json", "The name of the output configuration file to write.")
	workDir       = flag.String("work_dir", "", "The working directory for the configgen.  If not set, defaults to the directory where the configgen is run.")
	version       = flag.String("version", "0.0.0", "The installer version.")
	suggestedUser = flag.String("suggested_user", "", "The user to run the installation as.")
	// TODO(filmil): Merge with configIn
	configFile = flag.String("config", "", "The file name containing the installer configuration.")
	uninstall  = flag.String("uninstall", "", "If set, the supplied clusters will be uninstalled.")
	useCurrent = flag.Bool("use_current_context", false, "If set, and if the list of clusters in the install config is empty, use current context to install into.")
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

// noninteractiveMain is the full content of the noninteractive installer main()
// function.  This will be pared down in a few steps to just the most necessary
// things.
func noninteractiveMain() {
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
	i := installer.New(config, dir)

	if *uninstall != "" {
		if err := i.Uninstall(*uninstall); err != nil {
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

func main() {
	flag.Parse()

	// A simple tee off to the full installer binary.
	if !*interactive {
		noninteractiveMain()
		return
	}

	c := config.NewDefaultConfig()
	if *configIn != "" {
		file, err := os.Open(*configIn)
		if err != nil {
			glog.Exit(errors.Wrapf(err, "while opening: %q", *configIn))
		}
		c, err = config.Load(file)
		if err != nil {
			glog.Exit(errors.Wrapf(err, "while loading: %q", *configIn))
		}
		if *suggestedUser != "" {
			c.User = *suggestedUser
		}
	}

	dir := path.Dir(os.Args[0])
	if *workDir != "" {
		dir = *workDir
	}
	v := versionOrDie(*version)
	glog.V(3).Infof("Using version: %v", v)
	g := configgen.New(v, dir, c, *configOut)

	if err := g.Run(); err != nil {
		glog.Exit(errors.Wrapf(err, "configgen reported an error"))
	}
}
