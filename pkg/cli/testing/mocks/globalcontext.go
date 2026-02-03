package mocks

import (
	"fmt"
)

// GlobalContextResolver provides mock resolution for GlobalContextEntry lookups
type GlobalContextResolver struct {
	resolver *MockResolver
}

// NewGlobalContextResolver creates a new GlobalContextResolver
func NewGlobalContextResolver(resolver *MockResolver) *GlobalContextResolver {
	return &GlobalContextResolver{
		resolver: resolver,
	}
}

// Resolve looks up a GlobalContextEntry by name and returns the mock value
func (g *GlobalContextResolver) Resolve(name string) (interface{}, error) {
	if g.resolver == nil {
		return nil, fmt.Errorf("resolver not initialized")
	}
	return g.resolver.ResolveGlobalContext(name)
}

// Has checks if a GlobalContextEntry mock exists
func (g *GlobalContextResolver) Has(name string) bool {
	if g.resolver == nil {
		return false
	}
	return g.resolver.HasGlobalContext(name)
}
