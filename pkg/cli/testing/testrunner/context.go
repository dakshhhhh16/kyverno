// Package testrunner provides mock integration for CLI test execution.
// This file bridges the mock system with the Kyverno test command.
package testrunner

import (
	"fmt"

	"github.com/go-logr/logr"
	"github.com/kyverno/kyverno/cmd/cli/kubectl-kyverno/apis/v1alpha1"
	"github.com/kyverno/kyverno/pkg/cli/testing/mocks"
	"github.com/kyverno/kyverno/pkg/engine/context/loaders"
)

// TestContext holds the test execution context including mock resolvers
type TestContext struct {
	// MockResolver handles mock resolution for API calls and GlobalContext
	MockResolver *mocks.MockResolver
	// Logger for test execution
	Logger logr.Logger
	// Values from the test configuration
	Values *v1alpha1.ValuesSpec
}

// NewTestContext creates a new test context from values configuration
func NewTestContext(logger logr.Logger, values *v1alpha1.ValuesSpec) (*TestContext, error) {
	ctx := &TestContext{
		Logger: logger,
		Values: values,
	}

	// Initialize mock resolver if mocks are defined
	if values != nil && values.Mocks != nil {
		mockConfig := convertToMockConfig(values.Mocks)
		resolver, err := mocks.NewMockResolver(mockConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to create mock resolver: %w", err)
		}
		ctx.MockResolver = resolver
		logger.V(2).Info("initialized mock resolver",
			"apiCalls", len(values.Mocks.APICalls),
			"globalContext", len(values.Mocks.GlobalContext),
			"httpCalls", len(values.Mocks.HTTPCalls),
		)
	}

	return ctx, nil
}

// Close releases resources held by the test context
func (tc *TestContext) Close() error {
	if tc.MockResolver != nil {
		return tc.MockResolver.Close()
	}
	return nil
}

// HasMocks returns true if mock configuration is available
func (tc *TestContext) HasMocks() bool {
	return tc.MockResolver != nil
}

// GetMockStore returns the mock resolver as a MockStore interface
// This is used by the engine loaders
func (tc *TestContext) GetMockStore() loaders.MockStore {
	if tc.MockResolver == nil {
		return nil
	}
	return NewMockStoreAdapter(tc.MockResolver)
}

// convertToMockConfig converts v1alpha1.MockConfig to mocks.MockConfig
func convertToMockConfig(apiMocks *v1alpha1.MockConfig) *mocks.MockConfig {
	if apiMocks == nil {
		return nil
	}

	config := &mocks.MockConfig{}

	// Convert API call mocks
	for _, m := range apiMocks.APICalls {
		config.APICallMocks = append(config.APICallMocks, mocks.APICallMock{
			URLPath:  m.URLPath,
			Method:   m.Method,
			Response: m.Response,
		})
	}

	// Convert GlobalContext mocks
	for _, m := range apiMocks.GlobalContext {
		config.GlobalContextMocks = append(config.GlobalContextMocks, mocks.GlobalContextMock{
			Name:  m.Name,
			Value: m.Value,
		})
	}

	// Convert HTTP call mocks
	for _, m := range apiMocks.HTTPCalls {
		httpMock := mocks.HTTPCallMock{
			URL:    m.URL,
			Method: m.Method,
			Response: mocks.HTTPResponse{
				Status:  m.Response.Status,
				Headers: m.Response.Headers,
				Body:    m.Response.Body,
			},
		}
		if m.RequestMatcher != nil {
			httpMock.RequestMatcher = &mocks.RequestMatcher{
				Headers:     m.RequestMatcher.Headers,
				BodyPattern: m.RequestMatcher.BodyPattern,
			}
		}
		config.HTTPCallMocks = append(config.HTTPCallMocks, httpMock)
	}

	return config
}

// MockStoreAdapter adapts MockResolver to the MockStore interface used by engine loaders
type MockStoreAdapter struct {
	resolver *mocks.MockResolver
}

// NewMockStoreAdapter creates a new adapter
func NewMockStoreAdapter(resolver *mocks.MockResolver) *MockStoreAdapter {
	return &MockStoreAdapter{resolver: resolver}
}

// ResolveAPICall implements MockStore
func (a *MockStoreAdapter) ResolveAPICall(urlPath string, vars map[string]string) (interface{}, error) {
	return a.resolver.ResolveAPICall(urlPath, vars)
}

// ResolveGlobalContext implements MockStore
func (a *MockStoreAdapter) ResolveGlobalContext(name string) (interface{}, error) {
	return a.resolver.ResolveGlobalContext(name)
}

// HasGlobalContext implements MockStore
func (a *MockStoreAdapter) HasGlobalContext(name string) bool {
	return a.resolver.HasGlobalContext(name)
}
