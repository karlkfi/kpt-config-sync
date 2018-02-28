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

//Package bash contains the commands that we send to the bash shell.
package bash

import (
	"context"

	"github.com/google/stolos/pkg/toolkit/exec"
	"github.com/pkg/errors"
)

var (
	bashCmd = exec.RequireProgram("bash")
)

// RunOrDie will execute a bash script and panic if the script fails.
func RunOrDie(ctx context.Context, scriptPath string) {
	if err := exec.New().Run(ctx, bashCmd, "-c", scriptPath); err != nil {
		panic(errors.Wrapf(err, "Script %s exited non-zero", scriptPath))
	}
}
