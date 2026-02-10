package testrunner

import (
	"fmt"
	"strings"
)

// TestMode represents the testing fidelity mode
type TestMode string

const (
	// ModeFast uses Smart Mocks (enhanced fake client with 50+ resources)
	ModeFast TestMode = "fast"

	// ModeAccurate uses envtest (real etcd + API server)
	ModeAccurate TestMode = "accurate"
)

// ParseTestMode parses a string into a TestMode
func ParseTestMode(s string) (TestMode, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "fast", "f", "quick":
		return ModeFast, nil
	case "accurate", "a", "full", "envtest":
		return ModeAccurate, nil
	default:
		return "", fmt.Errorf("unknown test mode %q: valid modes are 'fast' or 'accurate'", s)
	}
}

// String returns the string representation of the mode
func (m TestMode) String() string {
	return string(m)
}

// Description returns a human-readable description of the mode
func (m TestMode) Description() string {
	switch m {
	case ModeFast:
		return "Fast Mode (Smart Mocks) - Quick policy checks with enhanced fake client"
	case ModeAccurate:
		return "Accurate Mode (envtest) - Deep testing with real API server"
	default:
		return "Unknown mode"
	}
}

// TestConfig holds the unified configuration for test execution
type TestConfig struct {
	// Mode selects between Fast and Accurate testing
	Mode TestMode

	// PolicyPaths contains paths to policy YAML files
	PolicyPaths []string

	// ResourcePaths contains paths to resource YAML files
	ResourcePaths []string

	// TargetResourcePaths contains paths to target resources (for mutate-existing)
	TargetResourcePaths []string

	// CRDPaths contains paths to Custom Resource Definition files
	CRDPaths []string

	// ExceptionPaths contains paths to policy exception files
	ExceptionPaths []string

	// VariablesPath is the path to the values/variables file
	VariablesPath string

	// UserInfoPath is the path to the user info file
	UserInfoPath string

	// RegistryAccess enables registry access for image verification
	RegistryAccess bool

	// Namespace sets the default namespace for namespaced resources
	Namespace string

	// FailFast stops execution on first failure
	FailFast bool

	// AutoFallback automatically falls back to Fast mode if Accurate mode
	// setup fails (e.g., envtest binaries not available)
	AutoFallback bool

	// EnvtestBinaryPath overrides the default envtest binary location
	EnvtestBinaryPath string
}

// DefaultConfig returns a TestConfig with sensible defaults
func DefaultConfig() TestConfig {
	return TestConfig{
		Mode:         ModeFast,
		Namespace:    "default",
		AutoFallback: true,
	}
}

// Validate checks the configuration for errors
func (c TestConfig) Validate() error {
	if c.Mode != ModeFast && c.Mode != ModeAccurate {
		return fmt.Errorf("invalid mode: %s", c.Mode)
	}
	if len(c.PolicyPaths) == 0 {
		return fmt.Errorf("at least one policy path is required")
	}
	if len(c.ResourcePaths) == 0 {
		return fmt.Errorf("at least one resource path is required")
	}
	return nil
}

// ModeCapabilities describes what each mode supports
type ModeCapabilities struct {
	Name                        TestMode
	SupportsCustomCRDs          bool
	SupportsAdmissionValidation bool
	SupportsRESTMapping         bool
	SupportsSchemaValidation    bool
	ResourceCount               int
	StartupTime                 string
}

// GetCapabilities returns the capabilities of a given mode
func GetCapabilities(mode TestMode) ModeCapabilities {
	switch mode {
	case ModeFast:
		return ModeCapabilities{
			Name:                        ModeFast,
			SupportsCustomCRDs:          true,
			SupportsAdmissionValidation: false,
			SupportsRESTMapping:         true,
			SupportsSchemaValidation:    false,
			ResourceCount:               50,
			StartupTime:                 "<100ms",
		}
	case ModeAccurate:
		return ModeCapabilities{
			Name:                        ModeAccurate,
			SupportsCustomCRDs:          true,
			SupportsAdmissionValidation: true,
			SupportsRESTMapping:         true,
			SupportsSchemaValidation:    true,
			ResourceCount:               -1,
			StartupTime:                 "2-5s",
		}
	default:
		return ModeCapabilities{}
	}
}
