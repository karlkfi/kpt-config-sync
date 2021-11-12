/*
Copyright 2021 Google LLC.

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

package kmetrics

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"sync"
	"time"
)

// RunKustomizeBuild runs `kustomize build` with the provided arguments.
// inputDir is the directory upon which to invoke `kustomize build`, and
// flags is what flags to run on it.
// For example: RunKustomizeBuild(".", []string{"--enable-helm"}) will run
// `kustomize build . --enable-helm`.
// The argument sendMetrics determines whether to send metrics about kustomize
// to Google Cloud.
func RunKustomizeBuild(ctx context.Context, sendMetrics bool, inputDir string, flags ...string) (string, error) {
	args := []string{"build", inputDir}
	args = append(args, flags...)
	cmd := exec.Command("kustomize", args...)
	return runKustomizeBuild(ctx, sendMetrics, inputDir, cmd)
}

// runKustomizeBuild will run `kustomize build` and also record measurements
// about kustomize usage via OpenCensus. This assumes that there is already an OC
// agent that is sending to a collector.
func runKustomizeBuild(ctx context.Context, sendMetrics bool, inputDir string, cmd *exec.Cmd) (string, error) {
	var wg sync.WaitGroup
	outputs := make(chan string, 1)
	errors := make(chan error, 1)
	wg.Add(1)

	go func() {
		executionTime, output, kustomizeErr := runCommand(cmd)
		// Send execution time and resource count metrics to OC collector
		resourceCount, err := kustomizeResourcesGenerated(output)
		if err == nil && sendMetrics {
			RecordKustomizeResourceCount(ctx, resourceCount)
			RecordKustomizeExecutionTime(ctx, float64(executionTime))
		}
		outputs <- output
		errors <- kustomizeErr
		wg.Done()
	}()

	kt, _ := readKustomizeFile(inputDir)
	fieldMetrics, fieldErr := kustomizeFieldUsage(kt, inputDir)
	if fieldErr == nil && sendMetrics {
		// Send field count metrics to OC collector
		RecordKustomizeFieldCountData(ctx, fieldMetrics)
	}

	wg.Wait()
	return <-outputs, <-errors
}

func runCommand(cmd *exec.Cmd) (int64, string, error) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	now := time.Now()
	err := cmd.Run()
	executionTime := time.Since(now).Nanoseconds()
	if err != nil {
		return executionTime, stdout.String(), fmt.Errorf(stderr.String())
	}
	return executionTime, stdout.String(), nil
}
