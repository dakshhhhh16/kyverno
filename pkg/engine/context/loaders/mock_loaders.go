// Package loaders provides context loaders for the Kyverno engine.
// This file adds mock-aware loaders for CLI testing.
package loaders

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/go-logr/logr"
	kyvernov1 "github.com/kyverno/kyverno/api/kyverno/v1"
	enginecontext "github.com/kyverno/kyverno/pkg/engine/context"
	"github.com/kyverno/kyverno/pkg/engine/jmespath"
)

// MockStore is an interface for resolving mock data during testing
type MockStore interface {
	// ResolveAPICall resolves a Kubernetes API call using mocks
	ResolveAPICall(urlPath string, vars map[string]string) (interface{}, error)
	// ResolveGlobalContext resolves a GlobalContextEntry using mocks
	ResolveGlobalContext(name string) (interface{}, error)
	// HasGlobalContext checks if a GlobalContext mock exists
	HasGlobalContext(name string) bool
}

// mockGctxLoader is a mock-aware global context loader for CLI testing
type mockGctxLoader struct {
	ctx       context.Context
	logger    logr.Logger
	entry     kyvernov1.ContextEntry
	enginectx enginecontext.Interface
	jp        jmespath.Interface
	mockStore MockStore
	data      []byte
}

// NewMockGCTXLoader creates a mock-aware global context loader
// If mockStore is nil, it falls back to the regular loader behavior
func NewMockGCTXLoader(
	ctx context.Context,
	logger logr.Logger,
	entry kyvernov1.ContextEntry,
	enginectx enginecontext.Interface,
	jp jmespath.Interface,
	mockStore MockStore,
) enginecontext.Loader {
	return &mockGctxLoader{
		ctx:       ctx,
		logger:    logger,
		entry:     entry,
		enginectx: enginectx,
		jp:        jp,
		mockStore: mockStore,
	}
}

func (g *mockGctxLoader) HasLoaded() bool {
	return g.data != nil
}

func (g *mockGctxLoader) LoadData() error {
	if g.entry.GlobalReference == nil {
		return fmt.Errorf("context entry does not have global reference")
	}

	gctxName := g.entry.GlobalReference.Name

	// Check if we have a mock for this GlobalContext entry
	if g.mockStore != nil && g.mockStore.HasGlobalContext(gctxName) {
		g.logger.V(4).Info("using mock data for GlobalContext", "name", gctxName)

		mockData, err := g.mockStore.ResolveGlobalContext(gctxName)
		if err != nil {
			return fmt.Errorf("failed to resolve mock GlobalContext %s: %w", gctxName, err)
		}

		jsonData, err := json.Marshal(mockData)
		if err != nil {
			return fmt.Errorf("failed to marshal mock data: %w", err)
		}

		if err := g.enginectx.AddContextEntry(g.entry.Name, jsonData); err != nil {
			return fmt.Errorf("failed to add mock context entry %s: %w", g.entry.Name, err)
		}

		g.data = jsonData
		g.logger.V(6).Info("added mock context data", "name", g.entry.Name)
		return nil
	}

	// No mock available - this is expected in test mode without mocks
	g.logger.V(4).Info("no mock found for GlobalContext, entry will be nil", "name", gctxName)
	return nil
}

// mockAPILoader is a mock-aware API call loader for CLI testing
type mockAPILoader struct {
	ctx       context.Context
	logger    logr.Logger
	entry     kyvernov1.ContextEntry
	enginectx enginecontext.Interface
	jp        jmespath.Interface
	mockStore MockStore
	data      []byte
}

// NewMockAPILoader creates a mock-aware API call loader
func NewMockAPILoader(
	ctx context.Context,
	logger logr.Logger,
	entry kyvernov1.ContextEntry,
	enginectx enginecontext.Interface,
	jp jmespath.Interface,
	mockStore MockStore,
) enginecontext.Loader {
	return &mockAPILoader{
		ctx:       ctx,
		logger:    logger,
		entry:     entry,
		enginectx: enginectx,
		jp:        jp,
		mockStore: mockStore,
	}
}

func (a *mockAPILoader) HasLoaded() bool {
	return a.data != nil
}

func (a *mockAPILoader) LoadData() error {
	if a.entry.APICall == nil {
		return fmt.Errorf("context entry does not have API call")
	}

	urlPath := a.entry.APICall.URLPath

	// Check if we have a mock for this API call
	if a.mockStore != nil {
		a.logger.V(4).Info("attempting to resolve mock API call", "urlPath", urlPath)

		mockData, err := a.mockStore.ResolveAPICall(urlPath, nil)
		if err == nil {
			// Mock found - use it
			jsonData, err := json.Marshal(mockData)
			if err != nil {
				return fmt.Errorf("failed to marshal mock API data: %w", err)
			}

			if err := a.enginectx.AddContextEntry(a.entry.Name, jsonData); err != nil {
				return fmt.Errorf("failed to add mock API context entry %s: %w", a.entry.Name, err)
			}

			a.data = jsonData
			a.logger.V(6).Info("added mock API call data", "name", a.entry.Name, "urlPath", urlPath)
			return nil
		}

		// No mock found - log and continue (test may fail if data is required)
		a.logger.V(4).Info("no mock found for API call", "urlPath", urlPath, "error", err)
	}

	return fmt.Errorf("no mock available for API call to %s", urlPath)
}
