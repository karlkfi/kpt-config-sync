package main

import (
	"flag"

	"github.com/golang/glog"
	"github.com/google/stolos/pkg/toolkit/dialog"
)

func main() {
	flag.Parse()
	glog.Infof("starting form")
	commonOpts := dialog.NewOptions(
		dialog.Title("Name Input"),
		dialog.Backtitle("Names"),
		dialog.Width(40),
		dialog.Height(11))

	f1 := "first"
	f2 := "last"
	f := dialog.NewForm(
		commonOpts,
		dialog.MenuHeight(3),
		dialog.Message("Please enter your first and last name."),
		dialog.FormItem(
			dialog.FormLabel{Text: "First name:", X: 1, Y: 1},
			dialog.FormLabel{Text: f1, X: 15, Y: 1},
			30, 40, &f1),
		dialog.FormItem(
			dialog.FormLabel{Text: "Last  name:", X: 1, Y: 3},
			dialog.FormLabel{Text: f2, X: 15, Y: 3},
			30, 40, &f2),
	)
	f.Display()
	glog.Infof("The menu dialog is non-blocking.")
	err := f.Close()
	glog.Infof("\nDone: f1: %q, f2: %q, err: %v\n", f1, f2, err)
}
