/*
Copyright 2017 The Kubernetes Authors.
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
	"github.com/mdruskin/kubernetes-enterprise-control/pkg/api/policyhierarchy/v1"
	"github.com/mdruskin/kubernetes-enterprise-control/pkg/client"
	"github.com/mdruskin/kubernetes-enterprise-control/pkg/service"
	"github.com/pkg/errors"
	core_v1 "k8s.io/api/core/v1"
	rbac_v1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var flagSyncDir = flag.String(
	"sync_dir", "", "The directory to deposit policy node files into.")
var flagUpdatePeriod = flag.Duration(
	"update_period", 5, "Frequency of updates expressed wtih time unit suffix (see time.ParseDuration)")
var flagMaxNamespaces = flag.Int(
	"max_namespaces", 20, "Max namespaces to create")

func main() {
	flag.Parse()

	if len(*flagSyncDir) == 0 {
		panic("Define --sync_dir")
	}

	Run(*flagSyncDir, *flagUpdatePeriod, *flagMaxNamespaces)
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
					Name:             name,
					WorkingNamespace: true,
					Policies: v1.PolicyLists{
						Roles: []rbac_v1.Role{
							rbac_v1.Role{
								ObjectMeta: metav1.ObjectMeta{
									Name: "pod-reader",
								},
								Rules: []rbac_v1.PolicyRule{
									rbac_v1.PolicyRule{
										Resources: []string{"pods"},
										Verbs:     []string{"get"},
									},
								},
							},
						},
						RoleBindings:   []rbac_v1.RoleBinding{},
						ResourceQuotas: []core_v1.ResourceQuotaSpec{},
					},
				}
				policyNode := client.WrapPolicyNodeSpec(nodeSpec)

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
