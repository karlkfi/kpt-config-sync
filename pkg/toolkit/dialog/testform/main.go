package main

import (
	"flag"

	"github.com/golang/glog"
	"github.com/google/nomos/pkg/toolkit/dialog"
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
			dialog.Label{Text: "First name:", X: 1, Y: 1},
			dialog.Field{
				Input: &f1,
				X:     15, Y: 1,
				ViewLen: 30,
				MaxLen:  40},
		),
		dialog.FormItem(
			dialog.Label{Text: "Last  name:", X: 1, Y: 3},
			dialog.Field{
				Input: &f2,
				X:     15, Y: 3,
				ViewLen: 30,
				MaxLen:  40},
		),
	)
	f.Display()
	glog.Infof("The menu dialog is non-blocking.")
	err := f.Close()
	glog.Infof("\nDone: f1: %q, f2: %q, err: %v\n", f1, f2, err)
}
