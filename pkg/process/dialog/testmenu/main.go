// Package testmenu contains a visual check for the menu binding.
package main

import (
	"flag"
	"fmt"

	"github.com/golang/glog"
	"github.com/google/nomos/pkg/process/dialog"
)

func main() {
	flag.Parse()
	glog.Infof("starting menu")
	commonOpts := dialog.NewOptions(
		dialog.Title("Menu Title"),
		dialog.Backtitle("Menu backtitle"),
		dialog.Width(40),
		dialog.Height(20),
		dialog.Colors(),
	)
	m := dialog.NewMenu(
		commonOpts,
		dialog.MenuHeight(5),
		dialog.Message("And now, \\Zb\\Z1please\\Zn select between...\nEven though you don't really need to"),
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
