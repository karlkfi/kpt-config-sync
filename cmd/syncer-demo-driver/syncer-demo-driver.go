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

// This binary is a test driver for demoing the syncer pipeline.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"path/filepath"
	"time"

	"github.com/golang/glog"
	"github.com/google/nomos/pkg/api/policyhierarchy/v1"
	"github.com/google/nomos/pkg/service"
	"github.com/google/nomos/pkg/testing/fakeorg"
	"github.com/google/nomos/pkg/testing/orgdriver"
	"github.com/google/nomos/pkg/util/policynode"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var flagSyncDir = flag.String(
	"sync_dir", "", "The directory to deposit policy node files into.")
var flagUpdatePeriod = flag.Duration(
	"update_period", time.Millisecond*2500, "Frequency of updates expressed with time unit suffix (see time.ParseDuration)")
var flagMaxNamespaces = flag.Int(
	"max_namespaces", 20, "Max namespaces to create")
var flagUseFakeOrg = flag.Bool(
	"use_fakeorg", false, "Use the fakeorg for generating stuff")
var flagGenFakeOrg = flag.Bool(
	"gen_fakeorg", false, "Generate namespaces for a fake org")
var flagRngSeed = flag.Int64(
	"rng_seed", 0, "Seed to use for RNG")

func main() {
	flag.Parse()

	if len(*flagSyncDir) == 0 {
		panic("Define --sync_dir")
	}

	switch {
	case *flagGenFakeOrg:
		GenFakeOrg(*flagSyncDir, *flagMaxNamespaces)
	case *flagUseFakeOrg:
		RunFakeOrg(*flagSyncDir, *flagUpdatePeriod, *flagMaxNamespaces)
	default:
		Run(*flagSyncDir, *flagUpdatePeriod, *flagMaxNamespaces)
	}
}

func setupDir(syncDir string) {
	err := os.MkdirAll(syncDir, 0750)
	if err != nil {
		panic(errors.Wrapf(err, "Failed to create sync dir %s", syncDir))
	}

	glog.Infof("Clearing sync dir %s", syncDir)
	fileInfos, err := ioutil.ReadDir(syncDir)
	if err != nil {
		panic(errors.Wrapf(err, "Failed to read sync dir %s", syncDir))
	}
	for _, fileInfo := range fileInfos {
		filePath := filepath.Join(syncDir, fileInfo.Name())
		glog.Infof("Deleting %s", filePath)
		err = os.Remove(filePath)
		if err != nil {
			panic(errors.Wrapf(err, "Failed to remove %s", filePath))
		}
	}
}

func createOptions(seed int64, numNamespaces int) *orgdriver.Options {
	if seed == 0 {
		seed = time.Now().UnixNano()
	}
	glog.Infof("Creating RNG with seed %d", seed)
	return orgdriver.NewOptions(rand.New(rand.NewSource(seed)), numNamespaces)
}

// GenFakeOrg generates a fake organization and writes it to the policy nodes.
func GenFakeOrg(syncDir string, numNamespaces int) {
	setupDir(syncDir)

	fakeOrg := fakeorg.New("my-fake-organization")
	options := createOptions(*flagRngSeed, numNamespaces)
	orgDriver := orgdriver.New(fakeOrg, options)
	orgDriver.GrowFakeOrg()
	orgDriver.WritePolicyNodes(syncDir)
}

// RunFakeOrg runs the driver using a fake organization.
func RunFakeOrg(syncDir string, updatePeriod time.Duration, maxNamespaces int) {
	setupDir(syncDir)

	fakeOrg := fakeorg.New("my-fake-organization")
	options := createOptions(*flagRngSeed, maxNamespaces)
	orgDriver := orgdriver.New(fakeOrg, options)

	ticker := time.NewTicker(updatePeriod)
	go func() {
		for range ticker.C {
			orgDriver.RandSteps(5)
			orgDriver.WriteDirtyPolicyNodes(syncDir)
		}
	}()

	service.WaitForShutdownSignal(ticker)
}

// Run is a loop that will randomly create / delete JSON formatted PolicyNode objects in the
// sync dir until the user sends sigint/sigterm.  Each policy node will be in its own file, so
// syncDir will be populated by a number of JSON files after running for a period of time.  Operations
// will proceed at a period defined by updatePeriod, and have a max namespace limit of maxNamespaces.
func Run(syncDir string, updatePeriod time.Duration, maxNamespaces int) {
	ticker := time.NewTicker(updatePeriod)

	err := os.MkdirAll(syncDir, 0750)
	if err != nil {
		panic(errors.Wrapf(err, "Failed to create sync dir %s", syncDir))
	}

	namespaceGeneratorNum := 0
	go func() {
		for range ticker.C {
			roll := rand.Float64()
			fileInfos, err := ioutil.ReadDir(syncDir)
			if err != nil {
				panic(errors.Wrapf(err, "Failed to read sync dir %s", syncDir))
			}

			// TODO: Add mutate operation
			createProbability := (float64(maxNamespaces) - float64(len(fileInfos))) / float64(maxNamespaces)
			isCreateOperation := (roll < createProbability)
			actions := map[bool]string{false: "delete", true: "create"}
			glog.Infof("Rolled %f, P(create)=%f, will %s", roll, createProbability, actions[isCreateOperation])

			if isCreateOperation {
				// create
				name := fmt.Sprintf("test-namespace-%d", namespaceGeneratorNum)
				filePath := filepath.Join(syncDir, fmt.Sprintf("%s.json", name))
				glog.Infof("Creating %s", filePath)
				namespaceGeneratorNum++

				nodeSpec := &v1.PolicyNodeSpec{
					Type: v1.Namespace,
					RolesV1: []rbacv1.Role{
						rbacv1.Role{
							ObjectMeta: metav1.ObjectMeta{
								Name: "pod-reader",
							},
							Rules: []rbacv1.PolicyRule{
								rbacv1.PolicyRule{
									Resources: []string{"pods"},
									Verbs:     []string{"get"},
								},
							},
						},
					},
					RoleBindingsV1:  []rbacv1.RoleBinding{},
					ResourceQuotaV1: &corev1.ResourceQuota{},
				}
				policyNode := policynode.NewPolicyNode(name, nodeSpec)

				policyNodeBytes, err := json.MarshalIndent(policyNode, "", "  ")
				if err != nil {
					panic(errors.Wrapf(err, "Failed to marshal struct"))
				}

				err = ioutil.WriteFile(filePath, policyNodeBytes, 0644)
				if err != nil {
					panic(errors.Wrapf(err, "Failed to write json bytes to %s", filePath))
				}
			} else {
				fileInfos, err := ioutil.ReadDir(syncDir)
				if err != nil {
					panic(errors.Wrapf(err, "Failed to read sync dir %s", syncDir))
				}

				if len(fileInfos) == 0 {
					glog.Info("No files to remove, nothing to do...")
					continue
				}

				removeIdx := rand.Int() % len(fileInfos)
				filePath := filepath.Join(syncDir, fileInfos[removeIdx].Name())
				glog.Infof("Deleting %s", filePath)
				err = os.Remove(filePath)
				if err != nil {
					panic(errors.Wrapf(err, "Failed to remove %s", filePath))
				}
			}
		}
	}()

	service.WaitForShutdownSignal(ticker)
}
