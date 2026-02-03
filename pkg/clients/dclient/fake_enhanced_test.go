package dclient

import (
	"testing"

	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestDefaultKubernetesResources(t *testing.T) {
	resources := defaultKubernetesResources()

	// Should have 50+ resources
	if len(resources) < 50 {
		t.Errorf("expected at least 50 resources, got %d", len(resources))
	}

	// Verify some key resources exist
	expectedResources := []schema.GroupVersionResource{
		{Version: "v1", Resource: "pods"},
		{Version: "v1", Resource: "configmaps"},
		{Version: "v1", Resource: "secrets"},
		{Version: "v1", Resource: "namespaces"},
		{Group: "apps", Version: "v1", Resource: "deployments"},
		{Group: "apps", Version: "v1", Resource: "statefulsets"},
		{Group: "batch", Version: "v1", Resource: "jobs"},
		{Group: "networking.k8s.io", Version: "v1", Resource: "networkpolicies"},
		{Group: "rbac.authorization.k8s.io", Version: "v1", Resource: "clusterroles"},
	}

	resourceSet := make(map[string]bool)
	for _, r := range resources {
		resourceSet[r.String()] = true
	}

	for _, expected := range expectedResources {
		if !resourceSet[expected.String()] {
			t.Errorf("missing expected resource: %s", expected.String())
		}
	}
}

func TestNewEnhancedFakeDiscoveryClient(t *testing.T) {
	// Test with no additional resources
	client := NewEnhancedFakeDiscoveryClient(nil)
	if client == nil {
		t.Fatal("expected non-nil client")
	}

	// Should have default resources
	if len(client.registeredResources) < 50 {
		t.Errorf("expected at least 50 resources, got %d", len(client.registeredResources))
	}

	// Test with additional custom resources
	customResources := []schema.GroupVersionResource{
		{Group: "custom.io", Version: "v1", Resource: "mycustomresources"},
	}

	client2 := NewEnhancedFakeDiscoveryClient(customResources)
	if len(client2.registeredResources) <= len(client.registeredResources) {
		t.Error("custom resources should be added")
	}
}

func TestFakeDiscoveryClient_RegisterKyvernoResources(t *testing.T) {
	client := NewEnhancedFakeDiscoveryClient(nil)
	initialCount := len(client.registeredResources)

	client.RegisterKyvernoResources()

	if len(client.registeredResources) <= initialCount {
		t.Error("RegisterKyvernoResources should add Kyverno resources")
	}

	// Verify Kyverno resources were added
	resourceSet := make(map[string]bool)
	for _, r := range client.registeredResources {
		resourceSet[r.String()] = true
	}

	expectedKyvernoResources := []string{
		"kyverno.io/v1, Resource=clusterpolicies",
		"kyverno.io/v1, Resource=policies",
	}

	for _, expected := range expectedKyvernoResources {
		found := false
		for r := range resourceSet {
			if r == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("missing Kyverno resource: %s", expected)
		}
	}
}

func TestGetDefaultResourceCount(t *testing.T) {
	count := GetDefaultResourceCount()
	if count < 50 {
		t.Errorf("expected at least 50 default resources, got %d", count)
	}
}

func TestFakeDiscoveryClient_RegisterCustomResources(t *testing.T) {
	client := NewEnhancedFakeDiscoveryClient(nil)
	initialCount := len(client.registeredResources)

	customResources := []schema.GroupVersionResource{
		{Group: "myapp.io", Version: "v1", Resource: "widgets"},
		{Group: "myapp.io", Version: "v1beta1", Resource: "gadgets"},
	}

	client.RegisterCustomResources(customResources)

	expectedCount := initialCount + len(customResources)
	if len(client.registeredResources) != expectedCount {
		t.Errorf("expected %d resources after registration, got %d", expectedCount, len(client.registeredResources))
	}
}
