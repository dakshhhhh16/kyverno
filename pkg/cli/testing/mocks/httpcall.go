package mocks

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
)

// HTTPMockServer provides a mock HTTP server for testing HTTP calls in policies
type HTTPMockServer struct {
	server *httptest.Server
	mocks  []HTTPCallMock
}

// NewHTTPMockServer creates a new HTTP mock server with the given mocks
func NewHTTPMockServer(mocks []HTTPCallMock) (*HTTPMockServer, error) {
	mockServer := &HTTPMockServer{
		mocks: mocks,
	}

	mockServer.server = httptest.NewServer(http.HandlerFunc(mockServer.handler))

	return mockServer, nil
}

// handler processes incoming HTTP requests and returns mock responses
func (s *HTTPMockServer) handler(w http.ResponseWriter, r *http.Request) {
	// Find matching mock
	for _, mock := range s.mocks {
		if s.matches(mock, r) {
			// Set response headers
			for key, value := range mock.Response.Headers {
				w.Header().Set(key, value)
			}

			// Set default content-type if not specified
			if w.Header().Get("Content-Type") == "" {
				w.Header().Set("Content-Type", "application/json")
			}

			// Set status code
			status := mock.Response.Status
			if status == 0 {
				status = http.StatusOK
			}
			w.WriteHeader(status)

			// Write body
			w.Write([]byte(mock.Response.Body))
			return
		}
	}

	// No match found
	w.WriteHeader(http.StatusNotFound)
	w.Write([]byte(`{"error": "No mock found for request"}`))
}

// matches checks if a request matches a mock configuration
func (s *HTTPMockServer) matches(mock HTTPCallMock, r *http.Request) bool {
	// Check method
	if mock.Method != "" && !strings.EqualFold(mock.Method, r.Method) {
		return false
	}

	// Check URL - the mock URL can be a partial match
	requestURL := r.URL.String()
	if !strings.Contains(requestURL, mock.URL) && !strings.Contains(mock.URL, requestURL) {
		// Try matching just the path
		if !strings.Contains(r.URL.Path, mock.URL) {
			return false
		}
	}

	// Check request matcher if present
	if mock.RequestMatcher != nil {
		// Check headers
		for key, value := range mock.RequestMatcher.Headers {
			if r.Header.Get(key) != value {
				return false
			}
		}

		// Check body pattern if present
		if mock.RequestMatcher.BodyPattern != "" {
			body, err := io.ReadAll(r.Body)
			if err != nil {
				return false
			}
			// Restore body for potential re-reading
			r.Body = io.NopCloser(strings.NewReader(string(body)))

			if !strings.Contains(string(body), mock.RequestMatcher.BodyPattern) {
				return false
			}
		}
	}

	return true
}

// URL returns the base URL of the mock server
func (s *HTTPMockServer) URL() string {
	if s.server != nil {
		return s.server.URL
	}
	return ""
}

// Close shuts down the mock server
func (s *HTTPMockServer) Close() error {
	if s.server != nil {
		s.server.Close()
	}
	return nil
}

// AddMock adds a new mock to the server at runtime
func (s *HTTPMockServer) AddMock(mock HTTPCallMock) {
	s.mocks = append(s.mocks, mock)
}
