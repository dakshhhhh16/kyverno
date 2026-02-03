package mocks

import (
	"net/http"
	"reflect"
	"testing"
)

func TestMockResolver_ResolveGlobalContext(t *testing.T) {
	tests := []struct {
		name      string
		config    *MockConfig
		lookupKey string
		want      interface{}
		wantErr   bool
	}{
		{
			name: "existing entry",
			config: &MockConfig{
				GlobalContextMocks: []GlobalContextMock{
					{Name: "deployment-count", Value: 5},
				},
			},
			lookupKey: "deployment-count",
			want:      5,
			wantErr:   false,
		},
		{
			name: "missing entry",
			config: &MockConfig{
				GlobalContextMocks: []GlobalContextMock{},
			},
			lookupKey: "nonexistent",
			want:      nil,
			wantErr:   true,
		},
		{
			name: "complex value",
			config: &MockConfig{
				GlobalContextMocks: []GlobalContextMock{
					{Name: "cluster-info", Value: map[string]interface{}{
						"name":   "prod-cluster",
						"region": "us-west-2",
					}},
				},
			},
			lookupKey: "cluster-info",
			want: map[string]interface{}{
				"name":   "prod-cluster",
				"region": "us-west-2",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolver, err := NewMockResolver(tt.config)
			if err != nil {
				t.Fatalf("failed to create resolver: %v", err)
			}
			defer resolver.Close()

			got, err := resolver.ResolveGlobalContext(tt.lookupKey)
			if (err != nil) != tt.wantErr {
				t.Errorf("ResolveGlobalContext() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ResolveGlobalContext() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMockResolver_ResolveAPICall(t *testing.T) {
	tests := []struct {
		name    string
		config  *MockConfig
		urlPath string
		vars    map[string]string
		wantErr bool
	}{
		{
			name: "exact match",
			config: &MockConfig{
				APICallMocks: []APICallMock{
					{
						URLPath:  "/api/v1/namespaces/default/configmaps",
						Response: map[string]interface{}{"items": []interface{}{}},
					},
				},
			},
			urlPath: "/api/v1/namespaces/default/configmaps",
			vars:    nil,
			wantErr: false,
		},
		{
			name: "pattern match with variable",
			config: &MockConfig{
				APICallMocks: []APICallMock{
					{
						URLPath:  "/api/v1/namespaces/{{namespace}}/configmaps",
						Response: map[string]interface{}{"items": []interface{}{}},
					},
				},
			},
			urlPath: "/api/v1/namespaces/{{namespace}}/configmaps",
			vars:    map[string]string{"namespace": "prod"},
			wantErr: false,
		},
		{
			name: "no match",
			config: &MockConfig{
				APICallMocks: []APICallMock{},
			},
			urlPath: "/api/v1/namespaces/default/secrets",
			vars:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolver, err := NewMockResolver(tt.config)
			if err != nil {
				t.Fatalf("failed to create resolver: %v", err)
			}
			defer resolver.Close()

			_, err = resolver.ResolveAPICall(tt.urlPath, tt.vars)
			if (err != nil) != tt.wantErr {
				t.Errorf("ResolveAPICall() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestHTTPMockServer(t *testing.T) {
	mocks := []HTTPCallMock{
		{
			URL:    "/validate",
			Method: "POST",
			Response: HTTPResponse{
				Status: 200,
				Body:   `{"valid": true}`,
			},
		},
		{
			URL:    "/deny",
			Method: "POST",
			Response: HTTPResponse{
				Status: 403,
				Body:   `{"allowed": false, "reason": "denied by policy"}`,
			},
		},
	}

	server, err := NewHTTPMockServer(mocks)
	if err != nil {
		t.Fatalf("failed to create mock server: %v", err)
	}
	defer server.Close()

	// Test successful response
	t.Run("successful POST", func(t *testing.T) {
		resp, err := http.Post(server.URL()+"/validate", "application/json", nil)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			t.Errorf("expected status 200, got %d", resp.StatusCode)
		}
	})

	// Test 403 response
	t.Run("denied response", func(t *testing.T) {
		resp, err := http.Post(server.URL()+"/deny", "application/json", nil)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != 403 {
			t.Errorf("expected status 403, got %d", resp.StatusCode)
		}
	})

	// Test not found
	t.Run("not found", func(t *testing.T) {
		resp, err := http.Get(server.URL() + "/unknown")
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != 404 {
			t.Errorf("expected status 404, got %d", resp.StatusCode)
		}
	})
}

func TestBuildURLPath(t *testing.T) {
	tests := []struct {
		name       string
		apiVersion string
		resource   string
		namespace  string
		resName    string
		want       string
	}{
		{
			name:       "core API namespaced resource",
			apiVersion: "v1",
			resource:   "configmaps",
			namespace:  "default",
			resName:    "my-config",
			want:       "/api/v1/namespaces/default/configmaps/my-config",
		},
		{
			name:       "core API cluster resource",
			apiVersion: "v1",
			resource:   "namespaces",
			namespace:  "",
			resName:    "kube-system",
			want:       "/api/v1/namespaces/kube-system",
		},
		{
			name:       "apps API",
			apiVersion: "apps/v1",
			resource:   "deployments",
			namespace:  "prod",
			resName:    "my-app",
			want:       "/apis/apps/v1/namespaces/prod/deployments/my-app",
		},
		{
			name:       "list all",
			apiVersion: "v1",
			resource:   "pods",
			namespace:  "default",
			resName:    "",
			want:       "/api/v1/namespaces/default/pods",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildURLPath(tt.apiVersion, tt.resource, tt.namespace, tt.resName)
			if got != tt.want {
				t.Errorf("BuildURLPath() = %v, want %v", got, tt.want)
			}
		})
	}
}
