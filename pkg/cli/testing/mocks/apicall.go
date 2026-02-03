package mocks

import (
	"fmt"
	"strings"
)

// APICallResolver provides mock resolution for Kubernetes API calls
type APICallResolver struct {
	resolver *MockResolver
}

// NewAPICallResolver creates a new APICallResolver
func NewAPICallResolver(resolver *MockResolver) *APICallResolver {
	return &APICallResolver{
		resolver: resolver,
	}
}

// Resolve looks up an API call by URL path and returns the mock response
func (a *APICallResolver) Resolve(urlPath string, vars map[string]string) (interface{}, error) {
	if a.resolver == nil {
		return nil, fmt.Errorf("resolver not initialized")
	}
	return a.resolver.ResolveAPICall(urlPath, vars)
}

// BuildURLPath constructs a Kubernetes API URL path from components
func BuildURLPath(apiVersion, resource, namespace, name string) string {
	var path strings.Builder

	// Parse API version to determine if it's core API or named group
	parts := strings.Split(apiVersion, "/")
	if len(parts) == 1 {
		// Core API (v1)
		path.WriteString("/api/")
		path.WriteString(apiVersion)
	} else {
		// Named group (apps/v1, networking.k8s.io/v1, etc.)
		path.WriteString("/apis/")
		path.WriteString(apiVersion)
	}

	// Add namespace if present
	if namespace != "" {
		path.WriteString("/namespaces/")
		path.WriteString(namespace)
	}

	// Add resource
	path.WriteString("/")
	path.WriteString(resource)

	// Add name if present
	if name != "" {
		path.WriteString("/")
		path.WriteString(name)
	}

	return path.String()
}

// BuildListURLPath constructs a Kubernetes API URL path for list operations
func BuildListURLPath(apiVersion, resource, namespace string, labels map[string]string) string {
	path := BuildURLPath(apiVersion, resource, namespace, "")

	// Add label selector if present
	if len(labels) > 0 {
		var labelSelector strings.Builder
		first := true
		for k, v := range labels {
			if !first {
				labelSelector.WriteString(",")
			}
			labelSelector.WriteString(k)
			labelSelector.WriteString("=")
			labelSelector.WriteString(v)
			first = false
		}
		path += "?labelSelector=" + labelSelector.String()
	}

	return path
}
