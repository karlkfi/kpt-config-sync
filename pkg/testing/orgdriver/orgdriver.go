/*
Copyright 2017 The Stolos Authors.

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

// Package orgdriver handles making pseudo-random transforms to an org.
package orgdriver

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"

	"github.com/golang/glog"
	"github.com/pkg/errors"

	"github.com/google/stolos/pkg/testing/fakeorg"
)

// Operation is the type of operation that modified the fake node.
type Operation string

// Enumeration of operations on the fake nodes.
const (
	OperationAdd      = "Add"
	OperationReparent = "Reparent"
	OperationRemove   = "Remove"
)

// Options are the options for the org driver
type Options struct {
	RNG      *rand.Rand // The rng to use
	MaxNodes int        // Max number of nodes to create
}

func NewOptions(rng *rand.Rand, maxNodes int) *Options {
	return &Options{
		RNG:      rng,
		MaxNodes: 20,
	}
}

// dirtyNode represents a node that has changed in the last iteration
type dirtyNode struct {
	node      *fakeorg.Node // The modified node
	operation Operation     // The operation that modified the node.
}

// OrgDriver handles manipulating a fake org by making pseudo random edits on the structure.
type OrgDriver struct {
	fakeOrg    *fakeorg.FakeOrg // Fake org object
	rng        *rand.Rand       // RNG for operations on org
	maxNodes   int              // Max number of nodes in the fake org
	dirtyNodes []*dirtyNode     // tracks edits to nodes for writing out changes.
}

// New creates a new org driver with the given options.
func New(fakeOrg *fakeorg.FakeOrg, options *Options) *OrgDriver {
	return &OrgDriver{
		fakeOrg:  fakeOrg,
		rng:      options.RNG,
		maxNodes: options.MaxNodes,
	}
}

// pSlider returns 1 - (value / limit), this is used for determining p(action) to bias towards create
// when no nodes exist, and bias towards delete when we are at the limit.
func pSlider(value int, limit int) float64 {
	return (float64(limit) - float64(value)) / float64(limit)
}

// GrowFakeOrg will build a fake org
func (s *OrgDriver) GrowFakeOrg() {
	for s.fakeOrg.Len() < s.maxNodes {
		s.doCreate()
	}

	for i := 0; i < 100+s.rng.Intn(100); i++ {
		s.doReparent()
	}
}

// RandSteps advances [1, maxSteps] steps using the internal RNG.
func (s *OrgDriver) RandSteps(maxSteps int) {
	steps := s.rng.Intn(maxSteps) + 1
	glog.Infof("Stepping %d times", steps)
	for i := 0; i < steps; i++ {
		s.Step()
		s.fakeOrg.Validate()
	}
}

// Step performs one modification of the fake org.
func (s *OrgDriver) Step() {
	minReparentNodeCount := 3 // Three nodes is the lower bound for being able to reparent a node.
	reparentProbability := float64(0)
	nodeCount := s.fakeOrg.Len()
	if minReparentNodeCount <= nodeCount {
		// Cap reparent at .33, linear increase from min node count to half of max nodes
		if s.fakeOrg.Len() > s.maxNodes/2 {
			reparentProbability = .33
		} else {
			halfMax := s.maxNodes / 2
			reparentProbability = (.33 * pSlider(nodeCount-minReparentNodeCount, halfMax))
		}
	}
	createProbability := (1 - reparentProbability) * pSlider(nodeCount, s.maxNodes)
	deleteProbability := 1 - reparentProbability - createProbability

	roll := s.rng.Float64()
	glog.Infof("Rolled %f, pCreate: %f pDelete: %f pReparent: %f", roll, createProbability, deleteProbability, reparentProbability)
	if roll < createProbability {
		s.doCreate()
	} else if roll < createProbability+reparentProbability {
		s.doReparent()
	} else {
		s.doDelete()
	}
}

// Performs a create operation
func (s *OrgDriver) doCreate() {
	// select random node, add a child
	nodeNames := s.fakeOrg.NodeNames()
	parentNode := s.fakeOrg.GetNode(nodeNames[s.rng.Intn(len(nodeNames))])

	nodeName := ""
	for {
		nodeName = fmt.Sprintf("node-%d", s.rng.Int63())
		if !s.fakeOrg.Contains(nodeName) {
			break
		}
	}
	newNode := fakeorg.NewNode(nodeName)
	glog.Infof("Creating %s as child of %s", newNode.PolicyNode().Name, parentNode.PolicyNode().Name)
	s.fakeOrg.AddNode(parentNode, newNode)
	s.dirtyNodes = append(s.dirtyNodes, &dirtyNode{
		node:      newNode,
		operation: OperationAdd,
	})
}

// Performs a reparent operation
func (s *OrgDriver) doReparent() {
	// select random node, for all other nodes, check if node is descendant. If not, reparent, if not satisified, try again
	nodeNames := s.fakeOrg.NodeNames()

	for len(nodeNames) != 0 {
		idx := s.rng.Intn(len(nodeNames))
		childName := nodeNames[idx]

		childNode := s.fakeOrg.GetNode(childName)
		for _, parentName := range nodeNames {
			if parentName == childName {
				continue
			}
			parentNode := s.fakeOrg.GetNode(parentName)
			if childNode.IsDescendant(parentNode) {
				// glog.V(5).Infof("Can't reparent %s to %s, is descendant", childName, parentName)
				continue
			}

			glog.Infof("Reparenting %s (%s -> %s)", childName, childNode.PolicyNode().Spec.Parent, parentName)
			s.fakeOrg.ReparentNode(parentNode, childNode)
			s.dirtyNodes = append(s.dirtyNodes, &dirtyNode{
				node:      childNode,
				operation: OperationReparent,
			})
			return
		}

		nodeNames = append(nodeNames[0:idx], nodeNames[idx+1:]...)
	}
	glog.Error("Was unable to find any nodes to reparent!")
}

// Performs a delete operation.
func (s *OrgDriver) doDelete() {
	if s.fakeOrg.Len() == 0 {
		glog.Error("Cannot delete nodes if only one exists!")
		return
	}

	leaves := s.fakeOrg.RootNode().Leaves()
	leafNode := s.fakeOrg.GetNode(leaves[s.rng.Intn(len(leaves))])
	glog.Infof("Deleting node %s", leafNode.Name())
	s.fakeOrg.RemoveNode(leafNode)
	s.dirtyNodes = append(s.dirtyNodes, &dirtyNode{
		node:      leafNode,
		operation: OperationRemove,
	})
}

// WriteDirtyPolicyNodes will write the dirty policy node resource json files to outputDir
func (s *OrgDriver) WriteDirtyPolicyNodes(outputDir string) {
	for _, dNode := range s.dirtyNodes {
		node := dNode.node
		filePath := filepath.Join(outputDir, fmt.Sprintf("%s.json", node.Name()))
		if dNode.operation == OperationRemove {
			glog.Infof("Deleting %s", filePath)
			err := os.Remove(filePath)
			if err != nil {
				panic(errors.Wrapf(err, "Failed delete %s", filePath))
			}
		} else {
			s.writeNode(node, outputDir)
		}
	}
	s.dirtyNodes = []*dirtyNode{}
}

// WritePolicyNodes writes out all policy nodes for the
func (s *OrgDriver) WritePolicyNodes(outputDir string) {
	for _, node := range s.fakeOrg.Nodes() {
		s.writeNode(node, outputDir)
	}
}

// writeNode writes out the node and panics on error
func (s *OrgDriver) writeNode(node *fakeorg.Node, outputDir string) {
	err := node.Write(outputDir)
	if err != nil {
		panic(errors.Wrapf(err, "Failed to write node"))
	}
}
