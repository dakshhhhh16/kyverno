package mocks

import (
	"fmt"
	"regexp"
	"strings"
)

// MockResolver handles resolution of mocked values during testing
type MockResolver struct {
	config     *MockConfig
	apiCalls   map[string]interface{}
	globalCtx  map[string]interface{}
	httpServer *HTTPMockServer
}

// NewMockResolver creates a new MockResolver from the given configuration
func NewMockResolver(config *MockConfig) (*MockResolver, error) {
	if config == nil {
		config = &MockConfig{}
	}

	resolver := &MockResolver{
		config:    config,
		apiCalls:  make(map[string]interface{}),
		globalCtx: make(map[string]interface{}),
	}

	// Index API call mocks by URL pattern
	for _, mock := range config.APICallMocks {
		resolver.apiCalls[mock.URLPath] = mock.Response
	}

	// Index GlobalContext mocks by name
	for _, mock := range config.GlobalContextMocks {
		resolver.globalCtx[mock.Name] = mock.Value
	}

	// Start HTTP mock server if needed
	if len(config.HTTPCallMocks) > 0 {
		server, err := NewHTTPMockServer(config.HTTPCallMocks)
		if err != nil {
			return nil, fmt.Errorf("failed to start HTTP mock server: %w", err)
		}
		resolver.httpServer = server
	}

	return resolver, nil
}

// ResolveAPICall resolves a Kubernetes API call using mocks
func (r *MockResolver) ResolveAPICall(urlPath string, vars map[string]string) (interface{}, error) {
	// Substitute variables in URL path
	resolvedPath := r.substituteVariables(urlPath, vars)

	// Try exact match first
	if response, exists := r.apiCalls[resolvedPath]; exists {
		return response, nil
	}

	// Try pattern matching for parameterized URLs
	for pattern, response := range r.apiCalls {
		if r.matchesPattern(pattern, resolvedPath) {
			return response, nil
		}
	}

	return nil, fmt.Errorf("no mock found for API call: %s", resolvedPath)
}

// ResolveGlobalContext resolves a GlobalContextEntry using mocks
func (r *MockResolver) ResolveGlobalContext(name string) (interface{}, error) {
	if value, exists := r.globalCtx[name]; exists {
		return value, nil
	}
	return nil, fmt.Errorf("no mock found for GlobalContextEntry: %s", name)
}

// HasGlobalContext checks if a GlobalContext mock exists
func (r *MockResolver) HasGlobalContext(name string) bool {
	_, exists := r.globalCtx[name]
	return exists
}

// GetHTTPServerURL returns the URL of the mock HTTP server, or empty if not running
func (r *MockResolver) GetHTTPServerURL() string {
	if r.httpServer != nil {
		return r.httpServer.URL()
	}
	return ""
}

// substituteVariables replaces {{variable}} patterns with actual values
func (r *MockResolver) substituteVariables(template string, vars map[string]string) string {
	result := template
	for key, value := range vars {
		placeholder := fmt.Sprintf("{{%s}}", key)
		result = strings.ReplaceAll(result, placeholder, value)
	}
	return result
}

// matchesPattern checks if a URL matches a pattern (supports {{variable}} as wildcard)
func (r *MockResolver) matchesPattern(pattern, url string) bool {
	// Convert pattern to regex (support {{variable}} as wildcard)
	regexPattern := regexp.QuoteMeta(pattern)
	regexPattern = regexp.MustCompile(`\\\{\\\{[^}]+\\\}\\\}`).ReplaceAllString(regexPattern, "[^/]+")
	regex, err := regexp.Compile("^" + regexPattern + "$")
	if err != nil {
		return false
	}
	return regex.MatchString(url)
}

// Close closes the mock resolver and any associated resources
func (r *MockResolver) Close() error {
	if r.httpServer != nil {
		return r.httpServer.Close()
	}
	return nil
}

// AddAPICallMock adds an API call mock at runtime
func (r *MockResolver) AddAPICallMock(urlPath string, response map[string]interface{}) {
	r.apiCalls[urlPath] = response
}

// AddGlobalContextMock adds a GlobalContext mock at runtime
func (r *MockResolver) AddGlobalContextMock(name string, value interface{}) {
	r.globalCtx[name] = value
}
