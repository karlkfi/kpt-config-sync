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

// Package configgen contains the utility for generating configurations.
package main

import (
	"flag"
	"fmt"
	"os"
	"path"
	"regexp"

	"github.com/blang/semver"
	"github.com/golang/glog"
	"github.com/google/stolos/pkg/toolkit/configgen"
	"github.com/google/stolos/pkg/toolkit/installer/config"
	"github.com/pkg/errors"
)

var (
	configIn      = flag.String("config_in", "", "The default configuration file to load.")
	configOut     = flag.String("config_out", "generated_config.json", "The name of the output configuration file to write.")
	workDir       = flag.String("work_dir", "", "The working directory for the configgen.  If not set, defaults to the directory where the configgen is run.")
	version       = flag.String("version", "0.0.0", "The installer version.")
	suggestedUser = flag.String("suggested_user", "", "The user to run the installation as.")
)

// version parses vstr, which could be of the form "prefix1.2.3-blah+blah".
func versionOrDie(vstr string) semver.Version {
	// Rewind the regexp to the first digit so it can be parsed as a semver.
	var digitRe = regexp.MustCompile("[0-9]")
	i := digitRe.FindStringIndex(vstr)
	if len(i) == 0 || i[0] < 0 {
		glog.Fatal(errors.Errorf("while parsing --version=%v", vstr))
	}
	v, err := semver.Parse(vstr[i[0]:])
	if err != nil {
		glog.Fatal(errors.Wrapf(err, "while parsing --version=%v", vstr))
	}
	return v
}

func main() {
	fmt.Printf("args: %+v\n", os.Args)
	flag.Parse()

	c := config.DefaultConfig
	if *configIn != "" {
		file, err := os.Open(*configIn)
		if err != nil {
			glog.Fatal(errors.Wrapf(err, "while opening: %q", *configIn))
		}
		c, err = config.Load(file)
		if err != nil {
			glog.Fatal(errors.Wrapf(err, "while loading: %q", *configIn))
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
	g := configgen.New(v, dir, c, *configOut)

	if err := g.Run(); err != nil {
		glog.Fatal(errors.Wrapf(err, "configgen reported an error"))
	}
}
