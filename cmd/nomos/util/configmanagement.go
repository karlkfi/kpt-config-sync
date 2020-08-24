package util

import (
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
)

const (
	// ConfigManagementName is the name of the ConfigManagement object.
	ConfigManagementName = "config-management"
	// ConfigManagementResource is the config management resource
	ConfigManagementResource = "configmanagements"
)

// DynamicClient obtains a client based on the supplied REST config.  Can be overridden in tests.
var DynamicClient = dynamic.NewForConfig

// ConfigManagementClient wraps a dynamic resource interface for reading ConfigManagement resources.
type ConfigManagementClient struct {
	resInt dynamic.ResourceInterface
}

// NewConfigManagementClient returns a new ConfigManagementClient.
func NewConfigManagementClient(cfg *rest.Config) (*ConfigManagementClient, error) {
	cl, err := DynamicClient(cfg)
	if err != nil {
		return nil, err
	}
	gvr := v1.SchemeGroupVersion.WithResource(ConfigManagementResource)
	return &ConfigManagementClient{cl.Resource(gvr).Namespace("")}, nil
}

// NestedString returns the string value specified by the given path of field names.
func (c *ConfigManagementClient) NestedString(fields ...string) (string, error) {
	unstr, err := c.resInt.Get(ConfigManagementName, metav1.GetOptions{}, "")
	if err != nil {
		return "", err
	}

	val, _, err := unstructured.NestedString(unstr.UnstructuredContent(), fields...)
	if err != nil {
		return "", errors.Wrap(err, "internal error parsing ConfigManagement")
	}

	return val, nil
}

// NestedStringSlice returns the string slice specified by the given path of field names.
func (c *ConfigManagementClient) NestedStringSlice(fields ...string) ([]string, error) {
	unstr, err := c.resInt.Get(ConfigManagementName, metav1.GetOptions{}, "")
	if err != nil {
		return nil, err
	}

	vals, _, err := unstructured.NestedStringSlice(unstr.UnstructuredContent(), fields...)
	if err != nil {
		return nil, errors.Wrap(err, "internal error parsing ConfigManagement")
	}

	return vals, nil
}
