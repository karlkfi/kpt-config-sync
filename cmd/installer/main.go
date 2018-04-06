// Package installer contains the binary that installs Nomos to target
// clusters.
//
// TODO(filmil): Configgen and installer are now by and large the
// same.  Consider fusing them together in a single binary.
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
	configFile    = flag.String("config", "", "The file name containing the installer configuration.")
	workDir       = flag.String("work_dir", "", "The working directory for the installer.  If not set, defaults to the directory where the installer is run.")
	uninstall     = flag.String("uninstall", "", "If set, the supplied clusters will be uninstalled.")
	yes           = flag.Bool("yes", false, "If yes, means that the user wants to do a destructive operation.")
	useCurrent    = flag.Bool("use_current_context", false, "If set, and if the list of clusters in the install config is empty, use current context to install into.")
	suggestedUser = flag.String("suggested_user", "", "The user to run the installation as.")
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
		err = i.Uninstall(*yes)
	} else {
		err = i.Run(*useCurrent)
	}
	if err != nil {
		glog.Fatal(errors.Wrap(err, "installer reported an error"))
	}
	glog.Infof("installer completed.")
}
