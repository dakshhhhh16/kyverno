package dclient

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// defaultKubernetesResources returns all standard Kubernetes resources (50+)
// This comprehensive list enables testing of policies against any standard K8s resource
func defaultKubernetesResources() []schema.GroupVersionResource {
	return []schema.GroupVersionResource{
		// ==========================================
		// Core API (v1) - 16 resources
		// ==========================================
		{Version: "v1", Resource: "bindings"},
		{Version: "v1", Resource: "componentstatuses"},
		{Version: "v1", Resource: "configmaps"},
		{Version: "v1", Resource: "endpoints"},
		{Version: "v1", Resource: "events"},
		{Version: "v1", Resource: "limitranges"},
		{Version: "v1", Resource: "namespaces"},
		{Version: "v1", Resource: "nodes"},
		{Version: "v1", Resource: "persistentvolumeclaims"},
		{Version: "v1", Resource: "persistentvolumes"},
		{Version: "v1", Resource: "pods"},
		{Version: "v1", Resource: "replicationcontrollers"},
		{Version: "v1", Resource: "resourcequotas"},
		{Version: "v1", Resource: "secrets"},
		{Version: "v1", Resource: "serviceaccounts"},
		{Version: "v1", Resource: "services"},

		// ==========================================
		// Apps API (apps/v1) - 5 resources
		// ==========================================
		{Group: "apps", Version: "v1", Resource: "controllerrevisions"},
		{Group: "apps", Version: "v1", Resource: "daemonsets"},
		{Group: "apps", Version: "v1", Resource: "deployments"},
		{Group: "apps", Version: "v1", Resource: "replicasets"},
		{Group: "apps", Version: "v1", Resource: "statefulsets"},

		// ==========================================
		// Batch API (batch/v1) - 2 resources
		// ==========================================
		{Group: "batch", Version: "v1", Resource: "cronjobs"},
		{Group: "batch", Version: "v1", Resource: "jobs"},

		// ==========================================
		// Networking (networking.k8s.io/v1) - 3 resources
		// ==========================================
		{Group: "networking.k8s.io", Version: "v1", Resource: "ingressclasses"},
		{Group: "networking.k8s.io", Version: "v1", Resource: "ingresses"},
		{Group: "networking.k8s.io", Version: "v1", Resource: "networkpolicies"},

		// ==========================================
		// Storage (storage.k8s.io/v1) - 5 resources
		// ==========================================
		{Group: "storage.k8s.io", Version: "v1", Resource: "csidrivers"},
		{Group: "storage.k8s.io", Version: "v1", Resource: "csinodes"},
		{Group: "storage.k8s.io", Version: "v1", Resource: "csistoragecapacities"},
		{Group: "storage.k8s.io", Version: "v1", Resource: "storageclasses"},
		{Group: "storage.k8s.io", Version: "v1", Resource: "volumeattachments"},

		// ==========================================
		// RBAC (rbac.authorization.k8s.io/v1) - 4 resources
		// ==========================================
		{Group: "rbac.authorization.k8s.io", Version: "v1", Resource: "clusterrolebindings"},
		{Group: "rbac.authorization.k8s.io", Version: "v1", Resource: "clusterroles"},
		{Group: "rbac.authorization.k8s.io", Version: "v1", Resource: "rolebindings"},
		{Group: "rbac.authorization.k8s.io", Version: "v1", Resource: "roles"},

		// ==========================================
		// Autoscaling (autoscaling/v2) - 1 resource
		// ==========================================
		{Group: "autoscaling", Version: "v2", Resource: "horizontalpodautoscalers"},
		{Group: "autoscaling", Version: "v1", Resource: "horizontalpodautoscalers"},

		// ==========================================
		// Policy (policy/v1) - 2 resources
		// ==========================================
		{Group: "policy", Version: "v1", Resource: "poddisruptionbudgets"},
		{Group: "policy", Version: "v1beta1", Resource: "podsecuritypolicies"},

		// ==========================================
		// Certificates (certificates.k8s.io/v1) - 1 resource
		// ==========================================
		{Group: "certificates.k8s.io", Version: "v1", Resource: "certificatesigningrequests"},

		// ==========================================
		// Coordination (coordination.k8s.io/v1) - 1 resource
		// ==========================================
		{Group: "coordination.k8s.io", Version: "v1", Resource: "leases"},

		// ==========================================
		// Discovery (discovery.k8s.io/v1) - 1 resource
		// ==========================================
		{Group: "discovery.k8s.io", Version: "v1", Resource: "endpointslices"},

		// ==========================================
		// Node (node.k8s.io/v1) - 1 resource
		// ==========================================
		{Group: "node.k8s.io", Version: "v1", Resource: "runtimeclasses"},

		// ==========================================
		// Scheduling (scheduling.k8s.io/v1) - 1 resource
		// ==========================================
		{Group: "scheduling.k8s.io", Version: "v1", Resource: "priorityclasses"},

		// ==========================================
		// Admission Registration (admissionregistration.k8s.io/v1) - 2 resources
		// ==========================================
		{Group: "admissionregistration.k8s.io", Version: "v1", Resource: "mutatingwebhookconfigurations"},
		{Group: "admissionregistration.k8s.io", Version: "v1", Resource: "validatingwebhookconfigurations"},

		// ==========================================
		// API Extensions (apiextensions.k8s.io/v1) - 1 resource
		// ==========================================
		{Group: "apiextensions.k8s.io", Version: "v1", Resource: "customresourcedefinitions"},

		// ==========================================
		// API Registration (apiregistration.k8s.io/v1) - 1 resource
		// ==========================================
		{Group: "apiregistration.k8s.io", Version: "v1", Resource: "apiservices"},

		// ==========================================
		// Events (events.k8s.io/v1) - 1 resource
		// ==========================================
		{Group: "events.k8s.io", Version: "v1", Resource: "events"},

		// ==========================================
		// Flowcontrol (flowcontrol.apiserver.k8s.io/v1) - 2 resources
		// ==========================================
		{Group: "flowcontrol.apiserver.k8s.io", Version: "v1", Resource: "flowschemas"},
		{Group: "flowcontrol.apiserver.k8s.io", Version: "v1", Resource: "prioritylevelconfigurations"},
	}
}

// NewEnhancedFakeDiscoveryClient creates a fake discovery client with 50+ resource types
// This is suitable for comprehensive policy testing
func NewEnhancedFakeDiscoveryClient(additionalResources []schema.GroupVersionResource) *fakeDiscoveryClient {
	allResources := defaultKubernetesResources()
	allResources = append(allResources, additionalResources...)

	return &fakeDiscoveryClient{
		registeredResources: allResources,
	}
}

// GetDefaultResourceCount returns the count of default resources
func GetDefaultResourceCount() int {
	return len(defaultKubernetesResources())
}

// RegisterKyvernoResources adds Kyverno-specific CRDs to the discovery client
func (c *fakeDiscoveryClient) RegisterKyvernoResources() {
	kyvernoResources := []schema.GroupVersionResource{
		{Group: "kyverno.io", Version: "v1", Resource: "clusterpolicies"},
		{Group: "kyverno.io", Version: "v1", Resource: "policies"},
		{Group: "kyverno.io", Version: "v1", Resource: "clusteradmissionreports"},
		{Group: "kyverno.io", Version: "v1", Resource: "admissionreports"},
		{Group: "kyverno.io", Version: "v2", Resource: "updaterequests"},
		{Group: "kyverno.io", Version: "v2", Resource: "cleanuppolicies"},
		{Group: "kyverno.io", Version: "v2", Resource: "clustercleanuppolicies"},
		{Group: "kyverno.io", Version: "v2alpha1", Resource: "globalcontextentries"},
		{Group: "wgpolicyk8s.io", Version: "v1alpha2", Resource: "clusterpolicyreports"},
		{Group: "wgpolicyk8s.io", Version: "v1alpha2", Resource: "policyreports"},
	}

	c.registeredResources = append(c.registeredResources, kyvernoResources...)
}

// RegisterCustomResources adds custom CRDs to the discovery client
func (c *fakeDiscoveryClient) RegisterCustomResources(resources []schema.GroupVersionResource) {
	c.registeredResources = append(c.registeredResources, resources...)
}

// ServerResourcesForGroupVersion returns API resources for a specific group/version
// This method is essential for proper discovery behavior in tests
func ServerResourcesForGroupVersion(resources []schema.GroupVersionResource, groupVersion string) []schema.GroupVersionResource {
	var result []schema.GroupVersionResource

	gv, err := schema.ParseGroupVersion(groupVersion)
	if err != nil {
		return result
	}

	for _, res := range resources {
		if res.Group == gv.Group && res.Version == gv.Version {
			result = append(result, res)
		}
	}

	return result
}

// GetResourcesForGroupVersion returns all resources for a specific group/version
func (c *fakeDiscoveryClient) GetResourcesForGroupVersion(groupVersion string) []schema.GroupVersionResource {
	return ServerResourcesForGroupVersion(c.registeredResources, groupVersion)
}

// GetAllGroupVersions returns all unique group/versions from registered resources
func (c *fakeDiscoveryClient) GetAllGroupVersions() []schema.GroupVersion {
	seen := make(map[schema.GroupVersion]bool)
	var result []schema.GroupVersion

	for _, res := range c.registeredResources {
		gv := schema.GroupVersion{Group: res.Group, Version: res.Version}
		if !seen[gv] {
			seen[gv] = true
			result = append(result, gv)
		}
	}

	return result
}

// HasResource checks if a specific resource is registered
func (c *fakeDiscoveryClient) HasResource(gvr schema.GroupVersionResource) bool {
	for _, res := range c.registeredResources {
		if res.Group == gvr.Group && res.Version == gvr.Version && res.Resource == gvr.Resource {
			return true
		}
	}
	return false
}
