package testrunner

import (
	"bytes"
	"context"
	"testing"

	kyvernov1 "github.com/kyverno/kyverno/api/kyverno/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// ====================================================================
// Mode Configuration Tests
// ====================================================================

func TestParseTestMode(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    TestMode
		wantErr bool
	}{
		{"fast", "fast", ModeFast, false},
		{"fast-short", "f", ModeFast, false},
		{"fast-quick", "quick", ModeFast, false},
		{"fast-uppercase", "FAST", ModeFast, false},
		{"accurate", "accurate", ModeAccurate, false},
		{"accurate-short", "a", ModeAccurate, false},
		{"accurate-full", "full", ModeAccurate, false},
		{"accurate-envtest", "envtest", ModeAccurate, false},
		{"invalid", "invalid-mode", "", true},
		{"empty", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseTestMode(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseTestMode(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ParseTestMode(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestModeDescription(t *testing.T) {
	tests := []struct {
		mode TestMode
		want string
	}{
		{ModeFast, "Fast Mode (Smart Mocks) - Quick policy checks with enhanced fake client"},
		{ModeAccurate, "Accurate Mode (envtest) - Deep testing with real API server"},
		{TestMode("unknown"), "Unknown mode"},
	}
	for _, tt := range tests {
		t.Run(string(tt.mode), func(t *testing.T) {
			if got := tt.mode.Description(); got != tt.want {
				t.Errorf("Description() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestGetCapabilities(t *testing.T) {
	fastCaps := GetCapabilities(ModeFast)
	if fastCaps.ResourceCount != 50 {
		t.Errorf("Fast mode resource count = %d, want 50", fastCaps.ResourceCount)
	}
	if fastCaps.SupportsAdmissionValidation {
		t.Error("Fast mode should not support admission validation")
	}
	if !fastCaps.SupportsRESTMapping {
		t.Error("Fast mode should support REST mapping")
	}

	accurateCaps := GetCapabilities(ModeAccurate)
	if !accurateCaps.SupportsAdmissionValidation {
		t.Error("Accurate mode should support admission validation")
	}
	if !accurateCaps.SupportsSchemaValidation {
		t.Error("Accurate mode should support schema validation")
	}
}

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		config  TestConfig
		wantErr bool
	}{
		{
			"valid-fast",
			TestConfig{
				Mode:          ModeFast,
				PolicyPaths:   []string{"policy.yaml"},
				ResourcePaths: []string{"resource.yaml"},
			},
			false,
		},
		{
			"valid-accurate",
			TestConfig{
				Mode:          ModeAccurate,
				PolicyPaths:   []string{"policy.yaml"},
				ResourcePaths: []string{"resource.yaml"},
			},
			false,
		},
		{
			"missing-policy",
			TestConfig{
				Mode:          ModeFast,
				ResourcePaths: []string{"resource.yaml"},
			},
			true,
		},
		{
			"missing-resource",
			TestConfig{
				Mode:        ModeFast,
				PolicyPaths: []string{"policy.yaml"},
			},
			true,
		},
		{
			"invalid-mode",
			TestConfig{
				Mode:          TestMode("invalid"),
				PolicyPaths:   []string{"policy.yaml"},
				ResourcePaths: []string{"resource.yaml"},
			},
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// ====================================================================
// Enhanced Fake Discovery Tests
// ====================================================================

func TestEnhancedFakeDiscovery(t *testing.T) {
	disco := newEnhancedFakeDiscovery()

	// Verify 50+ resources are pre-registered
	count := disco.ResourceCount()
	if count < 50 {
		t.Errorf("Expected 50+ resources, got %d", count)
	}
	t.Logf("Enhanced discovery has %d pre-registered resources", count)

	// Verify core resources are findable
	coreKinds := []string{"Pod", "Service", "Deployment", "ConfigMap", "Secret",
		"Namespace", "Node", "DaemonSet", "StatefulSet", "Job", "CronJob",
		"Ingress", "NetworkPolicy", "ClusterRole", "ClusterRoleBinding",
		"ServiceAccount", "PersistentVolumeClaim", "StorageClass"}

	for _, kind := range coreKinds {
		gvr, found := disco.FindResource(kind)
		if !found {
			t.Errorf("Expected to find resource for kind %s", kind)
		} else {
			t.Logf("Found %s → %s", kind, gvr.String())
		}
	}

	// Verify Kyverno CRDs are registered
	kyvernoKinds := []string{"ClusterPolicy", "Policy", "UpdateRequest",
		"CleanupPolicy", "PolicyReport", "GlobalContextEntry"}
	for _, kind := range kyvernoKinds {
		_, found := disco.FindResource(kind)
		if !found {
			t.Errorf("Expected to find Kyverno resource for kind %s", kind)
		}
	}
}

func TestEnhancedFakeDiscoveryRegisterCustom(t *testing.T) {
	disco := newEnhancedFakeDiscovery()
	initialCount := disco.ResourceCount()

	// Register a custom CRD
	customGVR := gvr("example.com", "v1", "widgets")
	customGVK := gvk("example.com", "v1", "Widget")
	disco.RegisterResource(customGVR, customGVK)

	if disco.ResourceCount() != initialCount+1 {
		t.Errorf("Expected %d resources after adding custom, got %d", initialCount+1, disco.ResourceCount())
	}

	// Verify it's findable
	foundGVR, found := disco.FindResource("Widget")
	if !found {
		t.Error("Custom resource Widget not found")
	}
	if foundGVR != customGVR {
		t.Errorf("Found wrong GVR: %v", foundGVR)
	}

	// Verify duplicate registration doesn't add twice
	disco.RegisterResource(customGVR, customGVK)
	if disco.ResourceCount() != initialCount+1 {
		t.Errorf("Duplicate registration should not increase count")
	}
}

func TestEnhancedFakeDiscoveryListGroups(t *testing.T) {
	disco := newEnhancedFakeDiscovery()
	groups := disco.ListGroups()

	if len(groups) == 0 {
		t.Fatal("Expected non-empty group list")
	}

	// Verify some expected groups
	expectedGroups := []string{"", "apps", "batch", "networking.k8s.io",
		"rbac.authorization.k8s.io", "kyverno.io"}
	groupNames := make(map[string]bool)
	for _, g := range groups {
		groupNames[g.Name] = true
	}

	for _, eg := range expectedGroups {
		if !groupNames[eg] {
			t.Errorf("Expected group %q in list", eg)
		}
	}
	t.Logf("Found %d API groups", len(groups))
}

// ====================================================================
// Fast Backend Tests
// ====================================================================

func TestFastBackendSetupTeardown(t *testing.T) {
	ctx := context.Background()
	backend := newFastBackend()

	if backend.IsReady() {
		t.Error("Backend should not be ready before setup")
	}
	if backend.Mode() != ModeFast {
		t.Errorf("Expected ModeFast, got %s", backend.Mode())
	}

	// Setup with some objects
	objects := []runtime.Object{
		newUnstructuredPod("test-ns", "test-pod"),
	}

	err := backend.Setup(ctx, objects)
	if err != nil {
		t.Fatalf("Setup failed: %v", err)
	}

	if !backend.IsReady() {
		t.Error("Backend should be ready after setup")
	}
	if backend.Client() == nil {
		t.Error("Client should not be nil after setup")
	}

	// Teardown
	err = backend.Teardown(ctx)
	if err != nil {
		t.Fatalf("Teardown failed: %v", err)
	}

	if backend.IsReady() {
		t.Error("Backend should not be ready after teardown")
	}
}

func TestFastBackendEmptySetup(t *testing.T) {
	ctx := context.Background()
	backend := newFastBackend()

	err := backend.Setup(ctx, nil)
	if err != nil {
		t.Fatalf("Empty setup failed: %v", err)
	}

	if !backend.IsReady() {
		t.Error("Backend should be ready even with empty setup")
	}

	_ = backend.Teardown(ctx)
}

// ====================================================================
// Unified Runner Tests (Fast Mode)
// ====================================================================

func TestRunnerFastMode(t *testing.T) {
	ctx := context.Background()
	config := TestConfig{
		Mode:          ModeFast,
		PolicyPaths:   []string{"test-policy.yaml"},
		ResourcePaths: []string{"test-resource.yaml"},
	}

	runner := NewTestRunner(config)
	var buf bytes.Buffer
	runner.SetOutput(&buf)

	// Create a test policy with a validation rule
	pol := newTestClusterPolicy("require-labels", "check-team-label",
		[]string{"Pod"}, true, false, false)

	// Create a test pod resource
	pod := newUnstructuredPod("default", "test-pod")

	// Run the unified test
	summary, err := runner.Run(ctx, []kyvernov1.PolicyInterface{pol}, []*unstructured.Unstructured{pod})
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	// Verify results
	if summary.Mode != ModeFast {
		t.Errorf("Expected ModeFast, got %s", summary.Mode)
	}
	if len(summary.Results) == 0 {
		t.Error("Expected at least one result")
	}
	if summary.Results[0].Mode != ModeFast {
		t.Errorf("Result mode should be ModeFast, got %s", summary.Results[0].Mode)
	}
	if summary.SetupDuration == 0 {
		t.Error("Setup duration should be non-zero")
	}
	if summary.TotalDuration == 0 {
		t.Error("Total duration should be non-zero")
	}

	t.Logf("Fast mode completed in %v with %d results", summary.TotalDuration, len(summary.Results))
	t.Logf("Output:\n%s", buf.String())
}

func TestRunnerMultiplePoliciesAndResources(t *testing.T) {
	ctx := context.Background()
	config := TestConfig{
		Mode:          ModeFast,
		PolicyPaths:   []string{"test.yaml"},
		ResourcePaths: []string{"test.yaml"},
	}

	runner := NewTestRunner(config)
	var buf bytes.Buffer
	runner.SetOutput(&buf)

	// Create multiple policies
	policies := []kyvernov1.PolicyInterface{
		newTestClusterPolicy("require-labels", "check-team", []string{"Pod"}, true, false, false),
		newTestClusterPolicy("restrict-image", "verify-registry", []string{"Pod"}, true, false, false),
		newTestClusterPolicy("disallow-privileged", "check-privileged", []string{"Pod"}, true, false, false),
	}

	// Create multiple resources
	resources := []*unstructured.Unstructured{
		newUnstructuredPod("default", "web-app"),
		newUnstructuredPod("production", "api-server"),
		newUnstructuredPod("staging", "worker"),
	}

	summary, err := runner.Run(ctx, policies, resources)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	// 3 policies × 3 resources × 1 rule each = 9 results
	expectedResults := 9
	if len(summary.Results) != expectedResults {
		t.Errorf("Expected %d results, got %d", expectedResults, len(summary.Results))
	}

	t.Logf("Evaluated %d policy-resource combinations in %v", len(summary.Results), summary.TotalDuration)
}

func TestRunnerPolicyResourceMismatch(t *testing.T) {
	ctx := context.Background()
	config := TestConfig{
		Mode:          ModeFast,
		PolicyPaths:   []string{"test.yaml"},
		ResourcePaths: []string{"test.yaml"},
	}

	runner := NewTestRunner(config)
	var buf bytes.Buffer
	runner.SetOutput(&buf)

	// Policy targets Deployments, but resource is a Pod
	pol := newTestClusterPolicy("require-replicas", "check-replicas",
		[]string{"Deployment"}, true, false, false)

	pod := newUnstructuredPod("default", "test-pod")

	summary, err := runner.Run(ctx, []kyvernov1.PolicyInterface{pol}, []*unstructured.Unstructured{pod})
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	// Should have 1 result with "skip" status
	if len(summary.Results) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(summary.Results))
	}
	if summary.Results[0].Status != "skip" {
		t.Errorf("Expected skip status, got %s", summary.Results[0].Status)
	}
	if summary.Skip != 1 {
		t.Errorf("Expected 1 skip, got %d", summary.Skip)
	}
}

func TestRunnerMutationPolicy(t *testing.T) {
	ctx := context.Background()
	config := TestConfig{
		Mode:          ModeFast,
		PolicyPaths:   []string{"test.yaml"},
		ResourcePaths: []string{"test.yaml"},
	}

	runner := NewTestRunner(config)
	var buf bytes.Buffer
	runner.SetOutput(&buf)

	// Create a mutation policy
	pol := newTestClusterPolicy("add-default-labels", "add-labels",
		[]string{"Pod"}, false, true, false)

	pod := newUnstructuredPod("default", "test-pod")

	summary, err := runner.Run(ctx, []kyvernov1.PolicyInterface{pol}, []*unstructured.Unstructured{pod})
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	if len(summary.Results) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(summary.Results))
	}
	if summary.Results[0].Status != "pass" {
		t.Errorf("Expected pass status for mutation, got %s", summary.Results[0].Status)
	}
}

func TestRunnerGeneratePolicy(t *testing.T) {
	ctx := context.Background()
	config := TestConfig{
		Mode:          ModeFast,
		PolicyPaths:   []string{"test.yaml"},
		ResourcePaths: []string{"test.yaml"},
	}

	runner := NewTestRunner(config)
	var buf bytes.Buffer
	runner.SetOutput(&buf)

	// Create a generate policy
	pol := newTestClusterPolicy("generate-networkpolicy", "gen-netpol",
		[]string{"Namespace"}, false, false, true)

	ns := newUnstructuredNamespace("production")

	summary, err := runner.Run(ctx, []kyvernov1.PolicyInterface{pol}, []*unstructured.Unstructured{ns})
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	if len(summary.Results) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(summary.Results))
	}
	if summary.Results[0].Status != "pass" {
		t.Errorf("Expected pass status for generate, got %s", summary.Results[0].Status)
	}
}

// ====================================================================
// Auto-Fallback Tests
// ====================================================================

func TestRunnerAutoFallbackEnabled(t *testing.T) {
	ctx := context.Background()
	config := TestConfig{
		Mode:          ModeAccurate,
		PolicyPaths:   []string{"test.yaml"},
		ResourcePaths: []string{"test.yaml"},
		AutoFallback:  true,
	}

	runner := NewTestRunner(config)
	var buf bytes.Buffer
	runner.SetOutput(&buf)

	pol := newTestClusterPolicy("require-labels", "check-team",
		[]string{"Pod"}, true, false, false)
	pod := newUnstructuredPod("default", "test-pod")

	// This will likely fail to start envtest (no binaries in CI),
	// but with auto-fallback it should succeed via Fast mode
	summary, err := runner.Run(ctx, []kyvernov1.PolicyInterface{pol}, []*unstructured.Unstructured{pod})
	if err != nil {
		t.Fatalf("Run with auto-fallback failed: %v", err)
	}

	// Should have fallen back to fast mode
	if summary.FellBack {
		t.Log("Correctly fell back from Accurate to Fast mode")
		if summary.Mode != ModeFast {
			t.Errorf("After fallback, mode should be Fast, got %s", summary.Mode)
		}
	} else {
		// If envtest actually worked (unlikely in unit test), that's fine too
		t.Log("Accurate mode succeeded (envtest available)")
	}

	if len(summary.Results) == 0 {
		t.Error("Expected results even after fallback")
	}
}

// ====================================================================
// Result Comparison Tests
// ====================================================================

func TestCompareResults(t *testing.T) {
	// Simulate Fast mode results
	fastSummary := &TestSummary{
		Mode:          ModeFast,
		TotalDuration: 50 * 1000 * 1000, // 50ms
		Results: []TestResult{
			{PolicyName: "p1", RuleName: "r1", ResourceKind: "Pod", ResourceName: "pod1", Status: "pass"},
			{PolicyName: "p1", RuleName: "r2", ResourceKind: "Pod", ResourceName: "pod1", Status: "fail"},
			{PolicyName: "p2", RuleName: "r1", ResourceKind: "Pod", ResourceName: "pod1", Status: "pass"},
		},
		Pass: 2, Fail: 1,
	}

	// Simulate Accurate mode results (slightly different)
	accurateSummary := &TestSummary{
		Mode:          ModeAccurate,
		TotalDuration: 3000 * 1000 * 1000, // 3s
		Results: []TestResult{
			{PolicyName: "p1", RuleName: "r1", ResourceKind: "Pod", ResourceName: "pod1", Status: "pass"},
			{PolicyName: "p1", RuleName: "r2", ResourceKind: "Pod", ResourceName: "pod1", Status: "pass"}, // differs!
			{PolicyName: "p2", RuleName: "r1", ResourceKind: "Pod", ResourceName: "pod1", Status: "pass"},
		},
		Pass: 3, Fail: 0,
	}

	report := CompareResults(fastSummary, accurateSummary)

	if report.Matching != 2 {
		t.Errorf("Expected 2 matching, got %d", report.Matching)
	}
	if report.Divergent != 1 {
		t.Errorf("Expected 1 divergent, got %d", report.Divergent)
	}
	if len(report.Divergences) != 1 {
		t.Fatalf("Expected 1 divergence detail, got %d", len(report.Divergences))
	}
	if report.Divergences[0].FastStatus != "fail" || report.Divergences[0].AccurateStatus != "pass" {
		t.Errorf("Wrong divergence: fast=%s accurate=%s",
			report.Divergences[0].FastStatus, report.Divergences[0].AccurateStatus)
	}

	// Print report for visual check
	var buf bytes.Buffer
	report.PrintReport(&buf)
	t.Log(buf.String())
}

// ====================================================================
// Default Config Tests
// ====================================================================

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()
	if config.Mode != ModeFast {
		t.Errorf("Default mode should be Fast, got %s", config.Mode)
	}
	if config.Namespace != "default" {
		t.Errorf("Default namespace should be 'default', got %s", config.Namespace)
	}
	if !config.AutoFallback {
		t.Error("AutoFallback should be true by default")
	}
}

// ====================================================================
// InferResourceFromKind Tests
// ====================================================================

func TestInferResourceFromKind(t *testing.T) {
	tests := []struct {
		kind     string
		expected string
	}{
		{"Deployment", "deployments"},
		{"Pod", "pods"},
		{"Service", "services"},
		{"Policy", "policies"},
		{"Ingress", "ingresses"},
		{"Namespace", "namespaces"},
		{"NetworkPolicy", "networkpolicies"},
	}

	for _, tt := range tests {
		t.Run(tt.kind, func(t *testing.T) {
			got := inferResourceFromKind(tt.kind)
			if got != tt.expected {
				t.Errorf("inferResourceFromKind(%q) = %q, want %q", tt.kind, got, tt.expected)
			}
		})
	}
}

// ====================================================================
// Helpers
// ====================================================================

func newUnstructuredPod(namespace, name string) *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "Pod",
			"metadata": map[string]interface{}{
				"name":      name,
				"namespace": namespace,
			},
			"spec": map[string]interface{}{
				"containers": []interface{}{
					map[string]interface{}{
						"name":  "app",
						"image": "nginx:latest",
					},
				},
			},
		},
	}
}

func newUnstructuredNamespace(name string) *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "Namespace",
			"metadata": map[string]interface{}{
				"name": name,
			},
		},
	}
}

func newTestClusterPolicy(name, ruleName string, kinds []string, validate, mutate, generate bool) kyvernov1.PolicyInterface {
	rule := kyvernov1.Rule{
		Name: ruleName,
		MatchResources: kyvernov1.MatchResources{
			Any: kyvernov1.ResourceFilters{
				kyvernov1.ResourceFilter{
					ResourceDescription: kyvernov1.ResourceDescription{
						Kinds: kinds,
					},
				},
			},
		},
	}

	if validate {
		rule.Validation = &kyvernov1.Validation{
			Message: "validation check",
		}
	}
	if mutate {
		rule.Mutation = &kyvernov1.Mutation{
			PatchesJSON6902: `[{"op": "add", "path": "/metadata/labels/mutated", "value": "true"}]`,
		}
	}
	if generate {
		rule.Generation = &kyvernov1.Generation{
			GeneratePattern: kyvernov1.GeneratePattern{
				ResourceSpec: kyvernov1.ResourceSpec{
					Kind: "NetworkPolicy",
					Name: "default-deny",
				},
			},
		}
	}

	return &kyvernov1.ClusterPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: kyvernov1.Spec{
			Rules: []kyvernov1.Rule{rule},
		},
	}
}

func gvr(group, version, resource string) schema.GroupVersionResource {
	return schema.GroupVersionResource{Group: group, Version: version, Resource: resource}
}

func gvk(group, version, kind string) schema.GroupVersionKind {
	return schema.GroupVersionKind{Group: group, Version: version, Kind: kind}
}
