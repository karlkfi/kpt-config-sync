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
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/golang/glog"
	"github.com/pkg/errors"
)

var (
	// Used only in package private tests to inject a substitute binary.
	testBinary string = ""
	testArgs          = []string{}
	testEnv           = []string{}

	// Outputs to be copied out if this module is used in a test fixture.
	testOutput      io.Reader = nil
	testErrorOutput io.Reader = nil
	testError       error     = nil

	// The set of programs that have been registered through RequireProgram and
	// the respective result of path lookup.
	requiredPrograms map[string]error = map[string]error{}
)

// SetFakeOutputsForTest sets fake sources for output data of an exec'd
// subprocess, and err is the fake error to be reported.  Any may be nil.
// Set all to nil to turn off testing behavior.
func SetFakeOutputsForTest(stdout, stderr io.Reader, err error) {
	testOutput = stdout
	testErrorOutput = stderr
	testError = err
	testBinary = "/SetFakeOutputsForTest/fake"
	if stdout == nil && stderr == nil && err == nil {
		testBinary = ""
	}
}

// RequireProgram finds the program with the given name in the system path.
// Use CheckProgram once all required programs have been registered to get
// a detailed report of the missing programs.
func RequireProgram(program string) string {
	path, err := exec.LookPath(program)
	if err != nil {
		requiredPrograms[program] = errors.Wrapf(err, "exec.LookPath")
		return program
	}
	glog.V(5).Infof("Using binary: %v for: %v", path, program)
	return path
}

// CheckProgram returns an error detailing the programs registered via
// RequireProgram that were not found but were required.
func CheckProgram() error {
	if len(requiredPrograms) == 0 {
		return nil
	}
	output := []string{}
	for name, err := range requiredPrograms {
		output = append(output, fmt.Sprintf("%v: %v", name, err))
	}
	return errors.Errorf("some required programs were not found: %v", strings.Join(output, "\n"))
}

// Context represents the execution context used for executing commands.
type Context interface {
	// Start runs the command specified by args, but does not wait for it to
	// complete.  Use Wait() to get the result.
	Start(ctx context.Context, args ...string)

	// Wait waits for the command in the current context to complete, returning
	// its error, if any.
	Wait() error

	// Run starts the command specified by args and waits for it to complete.
	Run(ctx context.Context, args ...string) error

	// Success returns true if the command completed with success, or false in
	// case of failure.
	Success() bool
}

// New creates a new execution context.  The program's inputs and outputs are
// connected to standard files.
func New() Context {
	return NewRedirected(os.Stdin, os.Stdout, os.Stderr)
}

// NewRedirected creates a new execution context attaching the supplied
// streams to the inputs and outputs.
func NewRedirected(stdin io.Reader, stdout, stderr io.Writer) Context {
	if testErrorOutput != nil || testOutput != nil || testError != nil {
		return &fakeContext{
			stdout:          stdout,
			stderr:          stderr,
			fakeOutput:      testOutput,
			fakeErrorOutput: testErrorOutput,
			err:             testError,
			done:            make(chan struct{})}
	}
	return &cmdContext{stdin: stdin, stdout: stdout, stderr: stderr}
}

var _ Context = (*cmdContext)(nil)

// cmdContext is a Context that executes an actual command.
type cmdContext struct {
	stdin  io.Reader
	stdout io.Writer
	stderr io.Writer

	cmd *exec.Cmd
	err error
}

// getAllArgs prepends the test-injected arguments if a binary has been substituted in a test fixture.
func getAllArgs(args ...string) []string {
	var allArgs []string
	if testBinary != "" {
		allArgs = append(testArgs, args...)
	} else {
		allArgs = args
	}
	return allArgs
}

// Start implements Context.Start
func (c *cmdContext) Start(ctx context.Context, args ...string) {
	if c.err != nil {
		glog.V(8).Info("exec.Run(): skipped execution because of prior error.")
		return
	}
	if len(args) == 0 {
		panic(errors.Errorf("can not run command with 0 parameters"))
	}

	allArgs := getAllArgs(args...)
	glog.V(1).Infof("exec.Run(%+v)", allArgs)
	c.cmd = exec.CommandContext(ctx, allArgs[0], allArgs[1:]...)
	c.cmd.Stdin = c.stdin
	c.cmd.Stdout = c.stdout
	c.cmd.Stderr = c.stderr
	c.cmd.Env = testEnv
	c.err = c.cmd.Start()
	glog.V(5).Infof("exec.Run(%+v) => %v", args, c.err)
}

// Wait implements Context.Wait.
func (c *cmdContext) Wait() error {
	if c.err != nil {
		return c.err
	}
	return c.cmd.Wait()
}

// Run implements Context.Run.
func (c *cmdContext) Run(ctx context.Context, args ...string) error {
	c.Start(ctx, args...)
	c.err = c.Wait()
	if c.err != nil {
		if _, ok := c.err.(*exec.ExitError); !ok {
			panic(errors.Wrapf(c.err, "Failed to properly execute command %+v", args))
		}
	}
	return c.err
}

// Success implements Contest.Success.
func (c *cmdContext) Success() bool {
	if c.cmd == nil {
		panic(errors.Errorf("called Success without a valid command"))
	}
	if c.cmd.ProcessState == nil {
		panic(errors.Errorf("called Success while process still running"))
	}
	return c.cmd.ProcessState.Success()
}

var _ Context = (*fakeContext)(nil)

// fakeContext is a context that substitutes fake output instead of running
// commands.  Used through SetFakeOutputsForTest.
type fakeContext struct {
	// Streams to be used as output.
	stdout io.Writer
	stderr io.Writer

	// Streams to be used to supply over to output.
	fakeOutput      io.Reader
	fakeErrorOutput io.Reader

	// Closed when the underlying copier routine is done with copying.
	done chan struct{}

	// Returned as result of Run() or Wait().
	err error
}

func (f *fakeContext) copyAll(t string, w io.Writer, r io.Reader) {
	if r == nil {
		return
	}
	if _, err := io.Copy(w, r); err != nil {
		f.err = errors.Wrapf(err, "while copying: %v", t)
	}
}

// Start implements Context.Start
func (f *fakeContext) Start(unused context.Context, args ...string) {
	// The Start...Wait API assumes that the process writing outputs is executed
	// asynchronously.  Fake that API by running a goroutine so that the rest
	// of the program can proceed.
	go func() {
		defer close(f.done)
		f.copyAll("stdout", f.stdout, f.fakeOutput)
		f.copyAll("stderr", f.stderr, f.fakeErrorOutput)
	}()
}

// Wait implements Context.Wait
func (f *fakeContext) Wait() error {
	<-f.done
	return f.err
}

// Run implements Contest.Run
func (f *fakeContext) Run(ctx context.Context, args ...string) error {
	f.Start(ctx, args...)
	return f.Wait()
}

// Success implements Context.Success
func (f *fakeContext) Success() bool {
	return f.err == nil
}
