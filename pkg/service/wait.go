/*
Copyright 2017 The Nomos Authors.
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

package service

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/golang/glog"
)

type Stoppable interface {
	Stop()
}

type genericStopper struct {
	stop func()
}

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

// WaitForShutdownWithChannel will wait until SIGINT or SIGTERM signal arrives or stopChannel is
// closed then return.
func WaitForShutdownWithChannel(stopChannel chan struct{}) {
	// wait for signal
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	glog.Infof("Waiting for shutdown signal INT/TERM...")
	select {
	case s := <-c:
		glog.Infof("Got signal %v, shutting down", s)
	case <-stopChannel:
		glog.Info("Stop channel closed, shutting down")
	}
}
