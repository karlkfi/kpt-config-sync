/*
Copyright 2019 The Nomos Authors.

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

package transform

import (
	"time"

	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/google/nomos/pkg/policyimporter/analyzer/visitor"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	importTokenKey = "importToken"
	importTimeKey  = "importTime"
)

// GCPAnnotationVisitor sets a list of bespin-specific annotations on GCP resources.
type GCPAnnotationVisitor struct {
	// Copying is used for copying parts of the ast.Root tree and continuing underlying visitor iteration.
	*visitor.Copying

	// importToken and importTime indicate at what time and which commit the resources are imported by
	// the policy importer.
	importToken, importTime string
}

var _ ast.Visitor = &GCPAnnotationVisitor{}

// NewGCPAnnotationVisitor returns a new GCPAnnotationVisitor.
func NewGCPAnnotationVisitor() *GCPAnnotationVisitor {
	v := &GCPAnnotationVisitor{
		Copying: visitor.NewCopying(),
	}
	v.SetImpl(v)
	return v
}

// VisitRoot implements Visitor.
func (v *GCPAnnotationVisitor) VisitRoot(c *ast.Root) *ast.Root {
	v.importToken = c.ImportToken
	v.importTime = c.LoadTime.Format(time.RFC3339)
	newNode := v.Copying.VisitRoot(c)
	return newNode
}

// VisitClusterObject implements Visitor.
func (v *GCPAnnotationVisitor) VisitClusterObject(o *ast.ClusterObject) *ast.ClusterObject {
	newObject := v.Copying.VisitClusterObject(o)
	applyAnnotation(newObject.FileObject, importTokenKey, v.importToken)
	applyAnnotation(newObject.FileObject, importTimeKey, v.importTime)
	return newObject
}

// VisitObject implements Visitor.
func (v *GCPAnnotationVisitor) VisitObject(o *ast.NamespaceObject) *ast.NamespaceObject {
	newObject := v.Copying.VisitObject(o)
	applyAnnotation(newObject.FileObject, importTokenKey, v.importToken)
	applyAnnotation(newObject.FileObject, importTimeKey, v.importTime)
	return newObject
}

// applyAnnotations applies annotation "aKey":"aValue" to the object.
func applyAnnotation(fo ast.FileObject, aKey, aValue string) {
	metaObj := fo.Object.(metav1.Object)
	a := metaObj.GetAnnotations()
	if a == nil {
		a = map[string]string{}
		metaObj.SetAnnotations(a)
	}
	a[aKey] = aValue
}
