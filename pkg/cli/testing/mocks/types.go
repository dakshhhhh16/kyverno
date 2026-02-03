// Package mocks provides mock implementations for Kyverno CLI testing.
// This package enables mocking of API calls, GlobalContext entries, and HTTP calls
// during policy testing without requiring a real Kubernetes cluster.
package mocks

// MockConfig represents the complete mock configuration
type MockConfig struct {
	APICallMocks       []APICallMock       `json:"apiCalls,omitempty"`
	GlobalContextMocks []GlobalContextMock `json:"globalContext,omitempty"`
	HTTPCallMocks      []HTTPCallMock      `json:"httpCalls,omitempty"`
}

// APICallMock represents a mock for a Kubernetes API call
type APICallMock struct {
	// URLPath is the API URL path pattern (supports {{variable}} placeholders)
	URLPath string `json:"urlPath"`
	// Method is the HTTP method (GET, POST, etc.) - defaults to GET
	Method string `json:"method,omitempty"`
	// Response is the mock response data
	Response map[string]interface{} `json:"response"`
}

// GlobalContextMock represents a mock for GlobalContextEntry
type GlobalContextMock struct {
	// Name is the name of the GlobalContextEntry
	Name string `json:"name"`
	// Value is the mock value to return
	Value interface{} `json:"value"`
}

// HTTPCallMock represents a mock for external HTTP calls
type HTTPCallMock struct {
	// URL is the URL pattern to match
	URL string `json:"url"`
	// Method is the HTTP method (GET, POST, etc.)
	Method string `json:"method"`
	// RequestMatcher provides additional matching criteria
	RequestMatcher *RequestMatcher `json:"requestMatcher,omitempty"`
	// Response is the mock HTTP response
	Response HTTPResponse `json:"response"`
}

// RequestMatcher provides criteria for matching HTTP requests
type RequestMatcher struct {
	// Headers to match in the request
	Headers map[string]string `json:"headers,omitempty"`
	// BodyPattern is a regex pattern to match the request body
	BodyPattern string `json:"bodyPattern,omitempty"`
}

// HTTPResponse represents a mock HTTP response
type HTTPResponse struct {
	// Status is the HTTP status code
	Status int `json:"status"`
	// Headers are the response headers
	Headers map[string]string `json:"headers,omitempty"`
	// Body is the response body
	Body string `json:"body"`
}
