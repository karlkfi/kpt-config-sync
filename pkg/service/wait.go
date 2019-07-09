package service

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/golang/glog"
)

// Stoppable defines an interface for types that can be stopped.
type Stoppable interface {
	// Stop stops the objects running threads of execution.
	Stop()
}

type genericStopper struct {
	stop func()
}

// Stop implements Stoppable.
func (s *genericStopper) Stop() {
	s.stop()
}

// NewStopper creates a stoppable from a function
func NewStopper(callback func()) Stoppable {
	return &genericStopper{stop: callback}
}

// WaitForShutdownSignal will block until the process receives a SIGINT or SIGTERM.  This is
// convenient for daemon like processes that want to shut down cleanly.
func WaitForShutdownSignal(stoppable Stoppable) {
	// wait for signal
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	glog.Infof("Waiting for shutdown signal INT/TERM...")
	s := <-c
	glog.Infof("Got signal %v, shutting down", s)
	stoppable.Stop()
}

// WaitForShutdownSignalCb will block until the process receives a SIGINT or SIGTERM.  This is
// convenient for daemon like processes that want to shut down cleanly.
// Invokes callback after shutting down
func WaitForShutdownSignalCb(stopChannel chan<- struct{}) {
	WaitForShutdownSignal(
		NewStopper(func() { close(stopChannel) }))
}
