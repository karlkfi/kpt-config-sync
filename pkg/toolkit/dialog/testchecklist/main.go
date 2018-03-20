// Package testchecklist is a demo of the checklist functionality.
package main

import (
	"flag"
	"fmt"

	"github.com/golang/glog"
	"github.com/google/nomos/pkg/toolkit/dialog"
)

func main() {
	flag.Parse()
	glog.Infof("testing checklist")
	commonOpts := dialog.NewOptions(
		dialog.Title("Menu Title"),
		dialog.Backtitle("Menu backtitle"),
		dialog.Width(40),
		dialog.Height(20))
	c := dialog.NewChecklist(
		commonOpts,
		dialog.MenuHeight(5),
		dialog.Message("And now, please select between..."),
		dialog.ChecklistItem("one", "Foobar", true),
		dialog.ChecklistItem("two", "Barbara", true),
		dialog.ChecklistItem("", "Untitled option", false),
	)
	c.Display()
	fmt.Printf("The menu dialog is non-blocking.\n")
	sel, err := c.Close()
	fmt.Printf("\nDone: selection: %q, err: %v\n", sel, err)
}
