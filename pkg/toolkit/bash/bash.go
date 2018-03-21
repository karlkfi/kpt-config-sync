/*
Copyright 2018 The Nomos Authors.
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

//Package bash contains the commands that we send to the bash shell.
package bash

import (
	"context"

	"github.com/google/nomos/pkg/toolkit/exec"
	"github.com/pkg/errors"
)

var (
	bashCmd = exec.RequireProgram("bash")
)

// runWithEnv executes a bash script with the given environment.  Returns
// the environment acknowledged by exec, and error if any.
func runWithEnv(ctx context.Context, scriptPath string, env ...string) ([]string, error) {
	c := exec.New()
	c.SetEnv(env)
	if err := c.Run(ctx, bashCmd, "-c", scriptPath); err != nil {
		errors.Wrapf(err, "Script %s exited non-zero", scriptPath)
	}
	return c.Env(), nil
}

// RunWithEnv will execute a bash script with the given environment as
// "KEY=VALUE" strings, returning an error, if any.
func RunWithEnv(ctx context.Context, scriptPath string, env ...string) error {
	_, err := runWithEnv(ctx, scriptPath, env...)
	return err
}

// RunOrDie will execute a bash script and panic if the script fails.
func RunOrDie(ctx context.Context, scriptPath string, env ...string) {
	if err := RunWithEnv(ctx, scriptPath, env...); err != nil {
		panic(err)
	}
}
