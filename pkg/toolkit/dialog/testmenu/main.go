// Package testmenu contains a visual check for the menu binding.
package main

import (
	"flag"
	"fmt"

	"github.com/golang/glog"
	"github.com/google/stolos/pkg/toolkit/dialog"
)

func main() {
	flag.Parse()
	glog.Infof("starting menu")
	commonOpts := dialog.NewOptions(
		dialog.Title("Menu Title"),
		dialog.Backtitle("Menu backtitle"),
		dialog.Width(40),
		dialog.Height(20))
	m := dialog.NewMenu(
		commonOpts,
		dialog.MenuHeight(5),
		dialog.Message("And now, please select between..."),
		dialog.MenuItem("one", "Foobar"),
		dialog.MenuItem("two", "Barbara"),
		dialog.MenuItem("", "Untitled option"),
		dialog.MenuItem("", "Untitled option"),
		dialog.MenuItem("", "Untitled option"),
		dialog.MenuItem("", "Untitled option"),
		dialog.MenuItem("", "Untitled option"),
		dialog.MenuItem("", "Untitled option"),
	)
	m.Display()
	fmt.Printf("The menu dialog is non-blocking.\n")
	sel, err := m.Close()
	fmt.Printf("\nDone: selection: %q, err: %v\n", sel, err)
}
