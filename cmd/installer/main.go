// Package installer contains the binary that installs Nomos to target clusters.
package main

import (
	"flag"
	"fmt"
	"os"
	"path"

	"github.com/golang/glog"
	"github.com/google/nomos/pkg/installer"
	"github.com/google/nomos/pkg/installer/config"
	"github.com/pkg/errors"
)

var (
	configFile = flag.String("config", "", "The file name containing the installer configuration.")
	workDir    = flag.String("work_dir", "", "The working directory for the installer.  If not set, defaults to the directory where the installer is run.")
	uninstall  = flag.String("uninstall", "", "If set, the supplied clusters will be uninstalled.")
	yes        = flag.Bool("yes", false, "If yes, means that the user wants to do a destructive operation.")
)

func main() {
	fmt.Printf("args: %+v\n", os.Args)
	flag.Parse()

	if *configFile == "" {
		flag.Usage()
		os.Exit(1)
	}
	glog.Infof("starting installer.")

	file, err := os.Open(*configFile)
	if err != nil {
		glog.Fatal(errors.Wrapf(err, "while opening: %q", *configFile))
	}

	config, err := config.Load(file)
	if err != nil {
		glog.Fatal(errors.Wrapf(err, "while loading: %q", *configFile))
	}

	dir := path.Dir(os.Args[0])
	if *workDir != "" {
		dir = *workDir
	}
	i := installer.New(config, dir)
	if *uninstall != "" {
		err = i.Uninstall(*yes)
	} else {
		err = i.Run()
	}
	if err != nil {
		glog.Fatal(errors.Wrap(err, "installer reported an error"))
	}
	glog.Infof("installer completed.")
}
