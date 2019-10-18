package fake

import (
	v1 "k8s.io/api/core/v1"
)

// ContainerObject returns an initialized Container.
func ContainerObject(name string) *v1.Container {
	return &v1.Container{Name: name}
}
