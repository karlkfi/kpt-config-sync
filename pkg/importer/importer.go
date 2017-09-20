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

package importer

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"github.com/docker/machine/libmachine/log"
	"github.com/fsnotify/fsnotify"
	"github.com/golang/glog"
	"github.com/mdruskin/kubernetes-enterprise-control/pkg/api/policyhierarchy/v1"
	"github.com/mdruskin/kubernetes-enterprise-control/pkg/client"
	"github.com/mdruskin/kubernetes-enterprise-control/pkg/service"
	"github.com/pkg/errors"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ImporterOptions provides constructor options for Importer
type ImporterOptions struct {
	// Daemon controls if the importer will watch the dir instead of running once then exiting.
	Daemon bool
	// ConfigDirPath is the directory we will watch for changes to files.
	ConfigDirPath string
	// Client is the kubernetes cluster client.
	Client *client.Client
}

// Importer handles importing yaml or json files containing a PolicyNodeSpec into the kubernetes
// cluster.
type Importer struct {
	options ImporterOptions
}

// New will create a new importer with the given options
func New(options ImporterOptions) *Importer {
	return &Importer{options: options}
}

func (i *Importer) Run() error {
	// NOTE: There is a slight race condition here for daemon mode if someone modifies a file right
	// after we do the first sync.
	err := i.Synchronize()
	if err != nil {
		return errors.Wrapf(err, "Failed to perform initial sync")
	}

	if !i.options.Daemon {
		return nil
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return errors.Wrapf(err, "Failed to set up watcher")
	}

	glog.Infof("Watching %s for changes", i.options.ConfigDirPath)
	watcher.Add(i.options.ConfigDirPath)
	if err != nil {
		return errors.Wrapf(err, "Failed to set up watch on %s", i.options.ConfigDirPath)
	}

	stop := make(chan struct{})
	go func() {
		defer func() {
			glog.Info("Watcher goroutine exited.")
		}()
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					glog.Info("Event channel closed, exiting watcher loop")
					return
				}

				glog.Infof("Got event %#v", event)
				var runSyncer bool
				if event.Op&fsnotify.Create == fsnotify.Create {
					err := watcher.Add(event.Name)
					if err != nil {
						panic(errors.Wrapf(err, "Failed to add file %s", event.Name))
					}
					runSyncer = true
				}

				if event.Op&fsnotify.Remove == fsnotify.Remove ||
					event.Op&fsnotify.Write == fsnotify.Write {
					runSyncer = true
				}

				// TODO: Schedule a sync in the future to debounce bulk writes, likely need to have a separate
				// gothread handle performing the sync
				if runSyncer {
					tries := 3
					var err error
					for try := 0; try < tries; try++ {
						err = i.Synchronize()
						if err != nil {
							time.Sleep(50 * time.Millisecond)
							continue
						}
						break
					}
					if err != nil {
						panic(errors.Wrapf(err, "Failed to synchronize after %d tries", tries))
					}
				}

			case err, ok := <-watcher.Errors:
				if !ok {
					glog.Info("Error channel closed, exiting watcher loop")
					return
				}
				panic(errors.Wrapf(err, "Got watcher error, assuming unrecoverable."))

			case <-stop:
				glog.Info("Stop channel closed, exiting watcher loop")
				return
			}
		}
	}()

	service.WaitForShutdownSignalCb(func() {
		close(stop)
	})

	return nil
}

// Synchronize will synchronize the directory with the existing custom resources.
func (i *Importer) Synchronize() error {
	// Read all specs from the directory
	fileInfos, err := ioutil.ReadDir(i.options.ConfigDirPath)
	if err != nil {
		return errors.Wrapf(err, "Failed to read dir %s", i.options.ConfigDirPath)
	}

	incomingPolicyNodes := make(map[string]*v1.PolicyNode)
	for _, fileinfo := range fileInfos {
		if fileinfo.IsDir() {
			continue
		}

		glog.Infof("Processing %s", fileinfo.Name())
		filePath := filepath.Join(i.options.ConfigDirPath, fileinfo.Name())
		policyNode, err := i.HandleFile(filePath)
		if err != nil {
			if os.IsNotExist(err) {
				log.Infof("File %s does not exist, skipping", fileinfo.Name())
				continue
			}
			return errors.Wrapf(err, "Failed to handle file %s", filePath)
		}

		incomingPolicyNodes[policyNode.ObjectMeta.Name] = policyNode
	}

	// Diff disk spec from one read from server
	clusterPolicyList, err := i.options.Client.PolicyHierarchy().K8usV1().PolicyNodes().List(meta_v1.ListOptions{})
	if err != nil {
		return errors.Wrapf(err, "Failed to list policy from server")
	}
	clusterPolicyNodes := make(map[string]*v1.PolicyNode)
	for _, policyNode := range clusterPolicyList.Items {
		clusterPolicyNodes[policyNode.ObjectMeta.Name] = policyNode.DeepCopy()
	}

	// creations
	var updates []*Update
	for name, incomingPolicyNode := range incomingPolicyNodes {
		clusterPolicyNode, ok := clusterPolicyNodes[name]
		if !ok {
			// create it
			glog.Infof("%s needs creation, queueing for create", name)
			updates = append(updates, NewUpdate(CreatePolicy, incomingPolicyNode))
		} else {
			// check for modify
			if !reflect.DeepEqual(incomingPolicyNode.Spec, clusterPolicyNode.Spec) {
				updates = append(updates, NewUpdate(UpdatePolicy, incomingPolicyNode))
			}
		}
	}

	// deletions
	for name, policyNode := range clusterPolicyNodes {
		if incomingPolicyNodes[name] == nil {
			// delete it
			glog.Infof("%s not found in incoming policy, queueing for delete", name)
			updates = append(updates, NewUpdate(DeletePolicy, policyNode))
		}
	}

	// Apply changes
	err = i.applyUpdates(updates)
	if err != nil {
		return errors.Wrapf(err, "Failed to apply all updates")
	}
	return nil
}

// HandleFile will load PolicyNode from a file
func (i *Importer) HandleFile(filePath string) (*v1.PolicyNode, error) {
	policyNode := &v1.PolicyNode{}
	if strings.HasSuffix(filePath, ".yaml") {
		// TODO: handle yaml
		return nil, errors.Errorf("Yaml file type not yet supported, cannot process %s", filePath)
	}

	if strings.HasSuffix(filePath, ".json") {
		jsonData, err := ioutil.ReadFile(filePath)
		if err != nil {
			return nil, errors.Wrapf(err, "Failed to read %s", filePath)
		}

		err = json.Unmarshal(jsonData, &policyNode)
		if err != nil {
			return nil, errors.Wrapf(err, "Failed to decode %s content: %s", filePath, string(jsonData))
		}

		return policyNode, nil
	}

	return nil, errors.Errorf("Unhandled file %s found", filePath)
}

func (i *Importer) applyUpdates(updates []*Update) error {
	glog.Infof("Applying updates...")
	for idx, update := range updates {
		err := i.applyUpdate(update)
		if err != nil {
			return errors.Wrapf(err, "Failed to apply update %d", idx)
		}
	}
	return nil
}

func (i *Importer) applyUpdate(update *Update) error {
	policyNodesIface := i.options.Client.PolicyHierarchy().K8usV1().PolicyNodes()
	policyNode := update.PolicyNode
	resourceName := policyNode.ObjectMeta.Name
	var err error
	switch update.Operation {
	case DeletePolicy:
		glog.Infof("Deleting %s", resourceName)
		err = policyNodesIface.Delete(resourceName, &meta_v1.DeleteOptions{})
		if err != nil {
			return errors.Wrapf(err, "Failed to delete policy %s", resourceName)
		}

	case CreatePolicy:
		newPolicyNode, err := policyNodesIface.Create(policyNode)
		if err != nil {
			return errors.Wrapf(err, "Failed to create policy %s", resourceName)
		}
		glog.Infof("Created %s at resourceversion %s", resourceName, newPolicyNode.ResourceVersion)

	case UpdatePolicy:
		newPolicyNode, err := policyNodesIface.Update(policyNode)
		if err != nil {
			return errors.Wrapf(err, "Failed to update policy %s", resourceName)
		}
		glog.Infof("Modified %s at resourceversion %s", resourceName, newPolicyNode.ResourceVersion)
	}

	return nil
}
