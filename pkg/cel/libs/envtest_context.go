// Package libs provides CEL library implementations for Kyverno.
// This file implements an EnvTest-backed context provider for real K8s API calls.
package libs

import (
	"context"
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
)

// EnvTestContextProvider implements the Context interface using a real K8s API server
// via controller-runtime's envtest package. This enables testing policies against
// actual Kubernetes API behavior including CRD validation and server-side logic.
type EnvTestContextProvider struct {
	env    *envtest.Environment
	client client.Client
	mapper meta.RESTMapper
	cfg    *envtest.Environment
}

// NewEnvTestContextProvider creates a new context provider backed by envtest.
// crdPaths specifies directories containing CRD YAML files to install.
func NewEnvTestContextProvider(crdPaths []string) (*EnvTestContextProvider, error) {
	env := &envtest.Environment{
		CRDDirectoryPaths: crdPaths,
	}

	cfg, err := env.Start()
	if err != nil {
		return nil, fmt.Errorf("failed to start envtest: %w", err)
	}

	// Create a client with the test environment's config
	k8sClient, err := client.New(cfg, client.Options{})
	if err != nil {
		env.Stop()
		return nil, fmt.Errorf("failed to create k8s client: %w", err)
	}

	return &EnvTestContextProvider{
		env:    env,
		client: k8sClient,
	}, nil
}

// Stop shuts down the envtest API server
func (c *EnvTestContextProvider) Stop() error {
	if c.env != nil {
		return c.env.Stop()
	}
	return nil
}

// GetGlobalReference retrieves a global context entry by name
func (c *EnvTestContextProvider) GetGlobalReference(name, projection string) (any, error) {
	// In envtest mode, global references are not supported
	// Return nil to allow tests to proceed
	return nil, nil
}

// GetImageData retrieves image metadata
func (c *EnvTestContextProvider) GetImageData(image string) (map[string]any, error) {
	// EnvTest doesn't support image registry access
	return nil, fmt.Errorf("image data not available in envtest mode")
}

// ToGVR converts apiVersion and kind to GroupVersionResource
func (c *EnvTestContextProvider) ToGVR(apiVersion, kind string) (*schema.GroupVersionResource, error) {
	gv, err := schema.ParseGroupVersion(apiVersion)
	if err != nil {
		return nil, err
	}
	// Use simple pluralization - in production this would use REST mapper
	resource := strings.ToLower(kind) + "s"
	return &schema.GroupVersionResource{
		Group:    gv.Group,
		Version:  gv.Version,
		Resource: resource,
	}, nil
}

// ListResources lists resources from the envtest API server
func (c *EnvTestContextProvider) ListResources(apiVersion, resource, namespace string, labels map[string]string) (*unstructured.UnstructuredList, error) {
	gv, err := schema.ParseGroupVersion(apiVersion)
	if err != nil {
		return nil, err
	}

	list := &unstructured.UnstructuredList{}
	list.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   gv.Group,
		Version: gv.Version,
		Kind:    resource + "List", // Convention: resource + "List"
	})

	opts := []client.ListOption{}
	if namespace != "" {
		opts = append(opts, client.InNamespace(namespace))
	}
	if len(labels) > 0 {
		opts = append(opts, client.MatchingLabels(labels))
	}

	if err := c.client.List(context.Background(), list, opts...); err != nil {
		return nil, fmt.Errorf("failed to list resources: %w", err)
	}

	return list, nil
}

// GetResource retrieves a single resource from the envtest API server
func (c *EnvTestContextProvider) GetResource(apiVersion, resource, namespace, name string) (*unstructured.Unstructured, error) {
	gv, err := schema.ParseGroupVersion(apiVersion)
	if err != nil {
		return nil, err
	}

	obj := &unstructured.Unstructured{}
	obj.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   gv.Group,
		Version: gv.Version,
		Kind:    resource, // Assuming resource is actually the Kind here
	})

	key := client.ObjectKey{
		Namespace: namespace,
		Name:      name,
	}

	if err := c.client.Get(context.Background(), key, obj); err != nil {
		return nil, err
	}

	return obj, nil
}

// PostResource creates a resource in the envtest API server
func (c *EnvTestContextProvider) PostResource(apiVersion, resource, namespace string, data map[string]any) (*unstructured.Unstructured, error) {
	gv, err := schema.ParseGroupVersion(apiVersion)
	if err != nil {
		return nil, err
	}

	obj := &unstructured.Unstructured{Object: data}
	obj.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   gv.Group,
		Version: gv.Version,
		Kind:    resource,
	})
	if namespace != "" {
		obj.SetNamespace(namespace)
	}

	if err := c.client.Create(context.Background(), obj); err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	return obj, nil
}

// GenerateResources stores generated resources (for generate rules)
func (c *EnvTestContextProvider) GenerateResources(namespace string, dataList []map[string]any) error {
	for _, data := range dataList {
		obj := &unstructured.Unstructured{Object: data}
		if namespace != "" {
			obj.SetNamespace(namespace)
		}
		if err := c.client.Create(context.Background(), obj); err != nil {
			return fmt.Errorf("failed to generate resource: %w", err)
		}
	}
	return nil
}

// GetGeneratedResources returns resources created via GenerateResources
func (c *EnvTestContextProvider) GetGeneratedResources() []*unstructured.Unstructured {
	// In envtest mode, resources are created in the real API server
	// This method would need to track created resources separately if needed
	return nil
}

// ClearGeneratedResources clears the generated resources list
func (c *EnvTestContextProvider) ClearGeneratedResources() {
	// No-op in envtest mode - resources persist in the API server
}

// SetGenerateContext sets the trigger context for generate rules
func (c *EnvTestContextProvider) SetGenerateContext(polName, triggerName, triggerNamespace, triggerAPIVersion, triggerGroup, triggerKind, triggerUID string, restoreCache bool) {
	// Context is set but not used in envtest mode
}

// Ensure EnvTestContextProvider implements the same interface as FakeContextProvider
var _ interface {
	GetGlobalReference(string, string) (any, error)
	GetImageData(string) (map[string]any, error)
	ToGVR(string, string) (*schema.GroupVersionResource, error)
	ListResources(string, string, string, map[string]string) (*unstructured.UnstructuredList, error)
	GetResource(string, string, string, string) (*unstructured.Unstructured, error)
	PostResource(string, string, string, map[string]any) (*unstructured.Unstructured, error)
} = (*EnvTestContextProvider)(nil)
