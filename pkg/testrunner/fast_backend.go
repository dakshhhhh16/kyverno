package testrunner

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/kyverno/kyverno/pkg/clients/dclient"
	engineapi "github.com/kyverno/kyverno/pkg/engine/api"
	kubeutils "github.com/kyverno/kyverno/pkg/utils/kube"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic/fake"
	kubefake "k8s.io/client-go/kubernetes/fake"
)

// fastBackend implements TestBackend using Smart Mocks
// This provides the "Fast Mode" - zero startup time, enhanced fake client
// with 50+ pre-registered Kubernetes resource types
type fastBackend struct {
	client dclient.Interface
	disco  *enhancedFakeDiscovery
	ready  bool
}

// newFastBackend creates a new Fast Mode backend
func newFastBackend() *fastBackend {
	return &fastBackend{}
}

// Setup initializes the fast backend with near-zero startup time
func (b *fastBackend) Setup(ctx context.Context, objects []runtime.Object) error {
	start := time.Now()

	// Create enhanced discovery client with 50+ resource types
	b.disco = newEnhancedFakeDiscovery()

	// Build scheme and GVR map from objects
	s := runtime.NewScheme()
	gvrToListKind := map[schema.GroupVersionResource]string{}

	for _, obj := range objects {
		gvk := obj.GetObjectKind().GroupVersionKind()
		if gvk.Kind == "" {
			continue
		}
		s.AddKnownTypeWithName(gvk, &unstructured.Unstructured{})
		listGVK := gvk
		listGVK.Kind += "List"
		s.AddKnownTypeWithName(listGVK, &unstructured.UnstructuredList{})

		// Register resource in discovery
		resource := strings.ToLower(gvk.Kind) + "s"
		gvr := schema.GroupVersionResource{
			Group:    gvk.Group,
			Version:  gvk.Version,
			Resource: resource,
		}
		gvrToListKind[gvr] = gvk.Kind + "List"
		b.disco.RegisterResource(gvr, gvk)
	}

	// Convert objects to unstructured
	unstructuredObjects := make([]runtime.Object, 0, len(objects))
	for _, obj := range objects {
		u, err := kubeutils.ObjToUnstructured(obj)
		if err != nil {
			// Skip objects that can't be converted
			continue
		}
		unstructuredObjects = append(unstructuredObjects, u)
	}

	// Create dynamic client
	dyn := fake.NewSimpleDynamicClientWithCustomListKinds(s, gvrToListKind, unstructuredObjects...)
	kclient := kubefake.NewSimpleClientset(unstructuredObjects...)

	// Build the dclient
	dClient, err := dclient.NewClient(ctx, dyn, kclient, time.Hour, false, nil)
	if err != nil {
		return fmt.Errorf("failed to create fake client: %w", err)
	}

	// Set enhanced discovery
	fakeDiscoClient := dclient.NewFakeDiscoveryClient(b.disco.AllGVRs())
	for gvr, gvk := range b.disco.gvrToGVK {
		fakeDiscoClient.AddGVRToGVKMapping(gvr, gvk)
	}
	dClient.SetDiscovery(fakeDiscoClient)

	b.client = dClient
	b.ready = true

	elapsed := time.Since(start)
	_ = elapsed // Available for logging: fmt.Printf("Fast backend setup: %v\n", elapsed)
	return nil
}

// Teardown is a no-op for the fast backend
func (b *fastBackend) Teardown(ctx context.Context) error {
	b.client = nil
	b.ready = false
	return nil
}

// Client returns the fake dclient
func (b *fastBackend) Client() dclient.Interface {
	return b.client
}

// ConfigmapResolver returns a no-op resolver for fast mode
func (b *fastBackend) ConfigmapResolver() engineapi.ConfigmapResolver {
	return nil
}

// Mode returns ModeFast
func (b *fastBackend) Mode() TestMode {
	return ModeFast
}

// IsReady returns whether the backend is initialized
func (b *fastBackend) IsReady() bool {
	return b.ready
}

// enhancedFakeDiscovery provides a comprehensive resource registry
// This is the "Smart Mock" layer - it knows about 50+ Kubernetes resources
// without needing a real API server
type enhancedFakeDiscovery struct {
	resources []schema.GroupVersionResource
	gvrToGVK  map[schema.GroupVersionResource]schema.GroupVersionKind
}

func newEnhancedFakeDiscovery() *enhancedFakeDiscovery {
	d := &enhancedFakeDiscovery{
		gvrToGVK: make(map[schema.GroupVersionResource]schema.GroupVersionKind),
	}

	// Pre-register 50+ standard Kubernetes resources
	coreResources := []struct {
		group    string
		version  string
		resource string
		kind     string
	}{
		// Core API (v1) - 16 resources
		{"", "v1", "bindings", "Binding"},
		{"", "v1", "componentstatuses", "ComponentStatus"},
		{"", "v1", "configmaps", "ConfigMap"},
		{"", "v1", "endpoints", "Endpoints"},
		{"", "v1", "events", "Event"},
		{"", "v1", "limitranges", "LimitRange"},
		{"", "v1", "namespaces", "Namespace"},
		{"", "v1", "nodes", "Node"},
		{"", "v1", "persistentvolumeclaims", "PersistentVolumeClaim"},
		{"", "v1", "persistentvolumes", "PersistentVolume"},
		{"", "v1", "pods", "Pod"},
		{"", "v1", "replicationcontrollers", "ReplicationController"},
		{"", "v1", "resourcequotas", "ResourceQuota"},
		{"", "v1", "secrets", "Secret"},
		{"", "v1", "serviceaccounts", "ServiceAccount"},
		{"", "v1", "services", "Service"},

		// Apps API (apps/v1) - 5 resources
		{"apps", "v1", "controllerrevisions", "ControllerRevision"},
		{"apps", "v1", "daemonsets", "DaemonSet"},
		{"apps", "v1", "deployments", "Deployment"},
		{"apps", "v1", "replicasets", "ReplicaSet"},
		{"apps", "v1", "statefulsets", "StatefulSet"},

		// Batch API (batch/v1) - 2 resources
		{"batch", "v1", "cronjobs", "CronJob"},
		{"batch", "v1", "jobs", "Job"},

		// Networking (networking.k8s.io/v1) - 3 resources
		{"networking.k8s.io", "v1", "ingressclasses", "IngressClass"},
		{"networking.k8s.io", "v1", "ingresses", "Ingress"},
		{"networking.k8s.io", "v1", "networkpolicies", "NetworkPolicy"},

		// Storage (storage.k8s.io/v1) - 5 resources
		{"storage.k8s.io", "v1", "csidrivers", "CSIDriver"},
		{"storage.k8s.io", "v1", "csinodes", "CSINode"},
		{"storage.k8s.io", "v1", "csistoragecapacities", "CSIStorageCapacity"},
		{"storage.k8s.io", "v1", "storageclasses", "StorageClass"},
		{"storage.k8s.io", "v1", "volumeattachments", "VolumeAttachment"},

		// RBAC (rbac.authorization.k8s.io/v1) - 4 resources
		{"rbac.authorization.k8s.io", "v1", "clusterrolebindings", "ClusterRoleBinding"},
		{"rbac.authorization.k8s.io", "v1", "clusterroles", "ClusterRole"},
		{"rbac.authorization.k8s.io", "v1", "rolebindings", "RoleBinding"},
		{"rbac.authorization.k8s.io", "v1", "roles", "Role"},

		// Autoscaling - 2 resources
		{"autoscaling", "v2", "horizontalpodautoscalers", "HorizontalPodAutoscaler"},
		{"autoscaling", "v1", "horizontalpodautoscalers", "HorizontalPodAutoscaler"},

		// Policy (policy/v1) - 1 resource
		{"policy", "v1", "poddisruptionbudgets", "PodDisruptionBudget"},

		// Certificates (certificates.k8s.io/v1) - 1 resource
		{"certificates.k8s.io", "v1", "certificatesigningrequests", "CertificateSigningRequest"},

		// Coordination (coordination.k8s.io/v1) - 1 resource
		{"coordination.k8s.io", "v1", "leases", "Lease"},

		// Discovery (discovery.k8s.io/v1) - 1 resource
		{"discovery.k8s.io", "v1", "endpointslices", "EndpointSlice"},

		// Node (node.k8s.io/v1) - 1 resource
		{"node.k8s.io", "v1", "runtimeclasses", "RuntimeClass"},

		// Scheduling (scheduling.k8s.io/v1) - 1 resource
		{"scheduling.k8s.io", "v1", "priorityclasses", "PriorityClass"},

		// Admission Registration - 2 resources
		{"admissionregistration.k8s.io", "v1", "mutatingwebhookconfigurations", "MutatingWebhookConfiguration"},
		{"admissionregistration.k8s.io", "v1", "validatingwebhookconfigurations", "ValidatingWebhookConfiguration"},

		// API Extensions - 1 resource
		{"apiextensions.k8s.io", "v1", "customresourcedefinitions", "CustomResourceDefinition"},

		// API Registration - 1 resource
		{"apiregistration.k8s.io", "v1", "apiservices", "APIService"},

		// Events - 1 resource
		{"events.k8s.io", "v1", "events", "Event"},

		// Flow Control - 2 resources
		{"flowcontrol.apiserver.k8s.io", "v1", "flowschemas", "FlowSchema"},
		{"flowcontrol.apiserver.k8s.io", "v1", "prioritylevelconfigurations", "PriorityLevelConfiguration"},

		// Kyverno CRDs - 10 resources
		{"kyverno.io", "v1", "clusterpolicies", "ClusterPolicy"},
		{"kyverno.io", "v1", "policies", "Policy"},
		{"kyverno.io", "v1", "clusteradmissionreports", "ClusterAdmissionReport"},
		{"kyverno.io", "v1", "admissionreports", "AdmissionReport"},
		{"kyverno.io", "v2", "updaterequests", "UpdateRequest"},
		{"kyverno.io", "v2", "cleanuppolicies", "CleanupPolicy"},
		{"kyverno.io", "v2", "clustercleanuppolicies", "ClusterCleanupPolicy"},
		{"kyverno.io", "v2alpha1", "globalcontextentries", "GlobalContextEntry"},
		{"wgpolicyk8s.io", "v1alpha2", "clusterpolicyreports", "ClusterPolicyReport"},
		{"wgpolicyk8s.io", "v1alpha2", "policyreports", "PolicyReport"},
	}

	for _, r := range coreResources {
		gvr := schema.GroupVersionResource{Group: r.group, Version: r.version, Resource: r.resource}
		gvk := schema.GroupVersionKind{Group: r.group, Version: r.version, Kind: r.kind}
		d.resources = append(d.resources, gvr)
		d.gvrToGVK[gvr] = gvk
	}

	return d
}

// RegisterResource adds a resource to the discovery
func (d *enhancedFakeDiscovery) RegisterResource(gvr schema.GroupVersionResource, gvk schema.GroupVersionKind) {
	// Don't add duplicates
	for _, existing := range d.resources {
		if existing == gvr {
			return
		}
	}
	d.resources = append(d.resources, gvr)
	d.gvrToGVK[gvr] = gvk
}

// AllGVRs returns all registered GVRs
func (d *enhancedFakeDiscovery) AllGVRs() []schema.GroupVersionResource {
	return d.resources
}

// ResourceCount returns the number of registered resources
func (d *enhancedFakeDiscovery) ResourceCount() int {
	return len(d.resources)
}

// FindResource finds a GVR by kind (case-insensitive)
func (d *enhancedFakeDiscovery) FindResource(kind string) (schema.GroupVersionResource, bool) {
	lowerKind := strings.ToLower(kind)
	for gvr, gvk := range d.gvrToGVK {
		if strings.ToLower(gvk.Kind) == lowerKind {
			return gvr, true
		}
	}
	return schema.GroupVersionResource{}, false
}

// ListGroups returns all unique API groups
func (d *enhancedFakeDiscovery) ListGroups() []metav1.APIGroup {
	seen := make(map[string]bool)
	var groups []metav1.APIGroup

	for _, gvr := range d.resources {
		groupName := gvr.Group
		if seen[groupName] {
			continue
		}
		seen[groupName] = true

		group := metav1.APIGroup{
			Name: groupName,
			Versions: []metav1.GroupVersionForDiscovery{
				{
					GroupVersion: gvr.Group + "/" + gvr.Version,
					Version:      gvr.Version,
				},
			},
			PreferredVersion: metav1.GroupVersionForDiscovery{
				GroupVersion: gvr.Group + "/" + gvr.Version,
				Version:      gvr.Version,
			},
		}
		groups = append(groups, group)
	}

	return groups
}
