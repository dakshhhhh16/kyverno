// Package envtest provides integration with controller-runtime's envtest
// for running tests against a real Kubernetes API server.
package envtest

import (
	"context"
	"fmt"
	"time"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
)

// Environment wraps controller-runtime's envtest.Environment
// with additional functionality for Kyverno CLI testing
type Environment struct {
	testEnv   *envtest.Environment
	cfg       *rest.Config
	client    client.Client
	scheme    *runtime.Scheme
	crdPaths  []string
	startTime time.Time
}

// Config holds configuration for the test environment
type Config struct {
	// CRDDirectoryPaths are paths to directories containing CRD YAML files
	CRDDirectoryPaths []string
	// BinaryAssetsDir is the directory containing envtest binaries
	BinaryAssetsDir string
	// UseExistingCluster uses an existing cluster instead of starting envtest
	UseExistingCluster bool
	// StartTimeout is the timeout for starting the environment
	StartTimeout time.Duration
	// StopTimeout is the timeout for stopping the environment
	StopTimeout time.Duration
	// Scheme is the runtime scheme to use
	Scheme *runtime.Scheme
}

// DefaultConfig returns a default configuration
func DefaultConfig() *Config {
	return &Config{
		StartTimeout: 60 * time.Second,
		StopTimeout:  60 * time.Second,
	}
}

// NewEnvironment creates a new test environment
func NewEnvironment(config *Config) (*Environment, error) {
	if config == nil {
		config = DefaultConfig()
	}

	env := &Environment{
		crdPaths: config.CRDDirectoryPaths,
		scheme:   config.Scheme,
	}

	useExisting := config.UseExistingCluster
	env.testEnv = &envtest.Environment{
		CRDDirectoryPaths:        config.CRDDirectoryPaths,
		BinaryAssetsDirectory:    config.BinaryAssetsDir,
		UseExistingCluster:       &useExisting,
		ControlPlaneStartTimeout: config.StartTimeout,
		ControlPlaneStopTimeout:  config.StopTimeout,
	}

	return env, nil
}

// Start starts the test environment
func (e *Environment) Start() error {
	e.startTime = time.Now()

	// Start the test environment
	cfg, err := e.testEnv.Start()
	if err != nil {
		return fmt.Errorf("failed to start envtest: %w", err)
	}
	e.cfg = cfg

	// Create Kubernetes client
	clientOpts := client.Options{}
	if e.scheme != nil {
		clientOpts.Scheme = e.scheme
	}

	e.client, err = client.New(cfg, clientOpts)
	if err != nil {
		e.testEnv.Stop()
		return fmt.Errorf("failed to create client: %w", err)
	}

	return nil
}

// Stop stops the test environment
func (e *Environment) Stop() error {
	if e.testEnv != nil {
		return e.testEnv.Stop()
	}
	return nil
}

// Client returns the Kubernetes client
func (e *Environment) Client() client.Client {
	return e.client
}

// Config returns the REST config
func (e *Environment) Config() *rest.Config {
	return e.cfg
}

// StartupTime returns the time it took to start the environment
func (e *Environment) StartupTime() time.Duration {
	return time.Since(e.startTime)
}

// InstallCRDs installs CRDs into the cluster
func (e *Environment) InstallCRDs(ctx context.Context, crds []*apiextensionsv1.CustomResourceDefinition) error {
	for _, crd := range crds {
		if err := e.client.Create(ctx, crd); err != nil {
			return fmt.Errorf("failed to create CRD %s: %w", crd.Name, err)
		}
	}

	// Wait for CRDs to be established
	return e.waitForCRDs(ctx, crds)
}

// waitForCRDs waits for CRDs to be established
func (e *Environment) waitForCRDs(ctx context.Context, crds []*apiextensionsv1.CustomResourceDefinition) error {
	for _, crd := range crds {
		if err := e.waitForCRD(ctx, crd.Name); err != nil {
			return err
		}
	}
	return nil
}

// waitForCRD waits for a single CRD to be established
func (e *Environment) waitForCRD(ctx context.Context, name string) error {
	timeout := time.After(30 * time.Second)
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timeout:
			return fmt.Errorf("timeout waiting for CRD %s to be established", name)
		case <-ticker.C:
			var crd apiextensionsv1.CustomResourceDefinition
			if err := e.client.Get(ctx, client.ObjectKey{Name: name}, &crd); err != nil {
				continue
			}
			for _, cond := range crd.Status.Conditions {
				if cond.Type == apiextensionsv1.Established && cond.Status == apiextensionsv1.ConditionTrue {
					return nil
				}
			}
		}
	}
}

// IsRunning returns true if the environment is running
func (e *Environment) IsRunning() bool {
	return e.cfg != nil
}
