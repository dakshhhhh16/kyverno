package testrunner

import (
	"testing"

	"github.com/go-logr/logr"
	"github.com/kyverno/kyverno/cmd/cli/kubectl-kyverno/apis/v1alpha1"
)

func TestNewTestContext_WithMocks(t *testing.T) {
	logger := logr.Discard()

	values := &v1alpha1.ValuesSpec{
		Mocks: &v1alpha1.MockConfig{
			GlobalContext: []v1alpha1.GlobalContextMock{
				{Name: "deployment-count", Value: 5},
				{Name: "cluster-name", Value: "test-cluster"},
			},
			APICalls: []v1alpha1.APICallMock{
				{
					URLPath: "/api/v1/namespaces/default/configmaps",
					Response: map[string]interface{}{
						"items": []interface{}{},
					},
				},
			},
		},
	}

	ctx, err := NewTestContext(logger, values)
	if err != nil {
		t.Fatalf("failed to create test context: %v", err)
	}
	defer ctx.Close()

	if !ctx.HasMocks() {
		t.Error("expected HasMocks() to return true")
	}

	// Test GlobalContext resolution
	adapter := NewMockStoreAdapter(ctx.MockResolver)

	if !adapter.HasGlobalContext("deployment-count") {
		t.Error("expected HasGlobalContext('deployment-count') to return true")
	}

	value, err := adapter.ResolveGlobalContext("deployment-count")
	if err != nil {
		t.Fatalf("failed to resolve GlobalContext: %v", err)
	}

	if value != 5 {
		t.Errorf("expected value 5, got %v", value)
	}

	// Test API call resolution
	apiResult, err := adapter.ResolveAPICall("/api/v1/namespaces/default/configmaps", nil)
	if err != nil {
		t.Fatalf("failed to resolve API call: %v", err)
	}

	if apiResult == nil {
		t.Error("expected non-nil API result")
	}
}

func TestNewTestContext_WithoutMocks(t *testing.T) {
	logger := logr.Discard()

	values := &v1alpha1.ValuesSpec{
		// No mocks defined
	}

	ctx, err := NewTestContext(logger, values)
	if err != nil {
		t.Fatalf("failed to create test context: %v", err)
	}
	defer ctx.Close()

	if ctx.HasMocks() {
		t.Error("expected HasMocks() to return false when no mocks defined")
	}
}

func TestNewTestContext_NilValues(t *testing.T) {
	logger := logr.Discard()

	ctx, err := NewTestContext(logger, nil)
	if err != nil {
		t.Fatalf("failed to create test context: %v", err)
	}
	defer ctx.Close()

	if ctx.HasMocks() {
		t.Error("expected HasMocks() to return false for nil values")
	}
}

func TestConvertToMockConfig(t *testing.T) {
	apiMocks := &v1alpha1.MockConfig{
		APICalls: []v1alpha1.APICallMock{
			{URLPath: "/api/v1/pods", Method: "GET", Response: map[string]interface{}{"items": []interface{}{}}},
		},
		GlobalContext: []v1alpha1.GlobalContextMock{
			{Name: "test", Value: "value"},
		},
		HTTPCalls: []v1alpha1.HTTPCallMock{
			{
				URL:    "/validate",
				Method: "POST",
				Response: v1alpha1.HTTPMockResponse{
					Status: 200,
					Body:   `{"valid": true}`,
				},
			},
		},
	}

	config := convertToMockConfig(apiMocks)

	if len(config.APICallMocks) != 1 {
		t.Errorf("expected 1 API call mock, got %d", len(config.APICallMocks))
	}

	if len(config.GlobalContextMocks) != 1 {
		t.Errorf("expected 1 GlobalContext mock, got %d", len(config.GlobalContextMocks))
	}

	if len(config.HTTPCallMocks) != 1 {
		t.Errorf("expected 1 HTTP call mock, got %d", len(config.HTTPCallMocks))
	}
}
