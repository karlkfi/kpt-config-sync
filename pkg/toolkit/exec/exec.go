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

// Package exec contains the wrappers for executing processes in a uniform
// way across Stolos.
package exec

import (
	"context"
	"io"
	"os"
	"os/exec"

	"github.com/golang/glog"
	"github.com/pkg/errors"
)

// RequireProgram find the program with the given name in the system path,
// or panics if it fails.
func RequireProgram(program string) string {
	path, err := exec.LookPath(program)
	if err != nil {
		glog.Fatalf("unexpected error: %v", err)
	}
	return path
}

// Context represents the execution context used for executing commands.
type Context struct {
	stdin  io.Reader
	stdout io.Writer
	stderr io.Writer

	cmd *exec.Cmd
	err error
}

// New creates a new execution context.  The program's inputs and outputs are
// connected to standard files.
func New() *Context {
	return NewRedirected(os.Stdin, os.Stdout, os.Stderr)
}

// NewRedirected creates a new execution context attaching the supplied
// streams to the inputs and outputs.
func NewRedirected(stdin io.Reader, stdout, stderr io.Writer) *Context {
	return &Context{stdin: stdin, stdout: stdout, stderr: stderr}
}

// Start runs the command specified by args, but does not wait for it to complete.
// Use Wait() to get the result.
func (c *Context) Start(ctx context.Context, args ...string) {
	if c.err != nil {
		glog.V(8).Info("exec.Run(): skipped execution because of prior error.")
		return
	}
	if len(args) == 0 {
		panic(errors.Errorf("can not run command with 0 parameters"))
	}

	glog.V(1).Infof("exec.Run(%+v)", args)
	c.cmd = exec.CommandContext(ctx, args[0], args[1:]...)
	c.cmd.Stdin = c.stdin
	c.cmd.Stdout = c.stdout
	c.cmd.Stderr = c.stderr
	c.err = c.cmd.Start()
	glog.V(5).Infof("exec.Run(%+v) => %v", args, c.err)
}

func (c *Context) Wait() error {
	if c.err != nil {
		return c.err
	}
	return c.cmd.Wait()
}

func (c *Context) Run(ctx context.Context, args ...string) error {
	c.Start(ctx, args...)
	c.err = c.Wait()
	if c.err != nil {
		if _, ok := c.err.(*exec.ExitError); !ok {
			panic(errors.Wrapf(c.err, "Failed to properly execute command %+v", args))
		}
	}
	return c.err
}

func (c *Context) Success() bool {
	if c.cmd == nil {
		panic(errors.Errorf("called Success without a valid command"))
	}
	if c.cmd.ProcessState == nil {
		panic(errors.Errorf("called Success while process still running"))
	}
	return c.cmd.ProcessState.Success()
}
