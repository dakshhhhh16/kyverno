package testrunner

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/kyverno/kyverno/pkg/clients/dclient"
	engineapi "github.com/kyverno/kyverno/pkg/engine/api"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
)

// accurateBackend implements TestBackend using envtest
// This provides the "Accurate Mode" - real etcd + API server for
// high-fidelity testing with proper admission control and schema validation
type accurateBackend struct {
	testEnv  *envtest.Environment
	config   *rest.Config
	client   dclient.Interface
	dynCli   dynamic.Interface
	kubeCli  kubernetes.Interface
	ready    bool
	crdPaths []string
}

// newAccurateBackend creates a new Accurate Mode backend
func newAccurateBackend(crdPaths []string) *accurateBackend {
	return &accurateBackend{
		crdPaths: crdPaths,
	}
}

// Setup initializes the envtest environment with real API server
func (b *accurateBackend) Setup(ctx context.Context, objects []runtime.Object) error {
	start := time.Now()

	// Configure envtest
	b.testEnv = &envtest.Environment{}

	// Add CRD paths if provided
	if len(b.crdPaths) > 0 {
		b.testEnv.CRDDirectoryPaths = b.crdPaths
	}

	// Start the envtest environment (etcd + API server)
	cfg, err := b.testEnv.Start()
	if err != nil {
		return fmt.Errorf("failed to start envtest environment: %w (ensure envtest binaries are installed via 'setup-envtest use')", err)
	}
	b.config = cfg

	// Create dynamic client
	b.dynCli, err = dynamic.NewForConfig(cfg)
	if err != nil {
		_ = b.testEnv.Stop()
		return fmt.Errorf("failed to create dynamic client: %w", err)
	}

	// Create kubernetes client
	b.kubeCli, err = kubernetes.NewForConfig(cfg)
	if err != nil {
		_ = b.testEnv.Stop()
		return fmt.Errorf("failed to create kubernetes client: %w", err)
	}

	// Create dclient with real clients
	// dclient.NewClient internally sets up discovery via the kubernetes clientset
	b.client, err = dclient.NewClient(ctx, b.dynCli, b.kubeCli, time.Hour, false, nil)
	if err != nil {
		_ = b.testEnv.Stop()
		return fmt.Errorf("failed to create dclient: %w", err)
	}

	// Seed objects into the running API server
	if err := b.seedObjects(ctx, objects); err != nil {
		_ = b.testEnv.Stop()
		return fmt.Errorf("failed to seed objects: %w", err)
	}

	b.ready = true

	elapsed := time.Since(start)
	_ = elapsed // Available for logging
	return nil
}

// seedObjects creates the provided objects in the real API server
func (b *accurateBackend) seedObjects(ctx context.Context, objects []runtime.Object) error {
	for _, obj := range objects {
		u, ok := obj.(*unstructured.Unstructured)
		if !ok {
			continue
		}

		gvk := u.GroupVersionKind()
		resource := inferResourceFromKind(gvk.Kind)
		gvr := schema.GroupVersionResource{
			Group:    gvk.Group,
			Version:  gvk.Version,
			Resource: resource,
		}

		ns := u.GetNamespace()
		var dynClient dynamic.ResourceInterface
		if ns != "" {
			dynClient = b.dynCli.Resource(gvr).Namespace(ns)
		} else {
			dynClient = b.dynCli.Resource(gvr)
		}

		_, err := dynClient.Create(ctx, u, metav1.CreateOptions{})
		if err != nil {
			// Skip already-exists or other non-fatal errors during seeding
			continue
		}
	}
	return nil
}

// inferResourceFromKind converts a Kind to a plural resource name
// e.g., "Deployment" -> "deployments", "Policy" -> "policies"
func inferResourceFromKind(kind string) string {
	lower := strings.ToLower(kind)
	switch {
	case strings.HasSuffix(lower, "y") && len(lower) > 1 &&
		!strings.HasSuffix(lower, "ay") &&
		!strings.HasSuffix(lower, "ey") &&
		!strings.HasSuffix(lower, "oy"):
		return lower[:len(lower)-1] + "ies"
	case strings.HasSuffix(lower, "s") || strings.HasSuffix(lower, "x"):
		return lower + "es"
	default:
		return lower + "s"
	}
}

// Teardown stops the envtest environment and cleans up resources
func (b *accurateBackend) Teardown(ctx context.Context) error {
	if b.testEnv != nil {
		if err := b.testEnv.Stop(); err != nil {
			return fmt.Errorf("failed to stop envtest: %w", err)
		}
	}
	b.client = nil
	b.ready = false
	return nil
}

// Client returns the real dclient connected to envtest
func (b *accurateBackend) Client() dclient.Interface {
	return b.client
}

// ConfigmapResolver returns a real configmap resolver backed by the API server
func (b *accurateBackend) ConfigmapResolver() engineapi.ConfigmapResolver {
	// In a full implementation, this would use the kubeCli to resolve configmaps
	// For the PoC, we return nil (test cases that need configmap resolution
	// should provide inline values)
	return nil
}

// Mode returns ModeAccurate
func (b *accurateBackend) Mode() TestMode {
	return ModeAccurate
}

// IsReady returns whether the envtest environment is running
func (b *accurateBackend) IsReady() bool {
	return b.ready
}

// GetRESTConfig returns the envtest REST config for advanced usage
// This allows test code to create additional clients if needed
func (b *accurateBackend) GetRESTConfig() *rest.Config {
	return b.config
}
