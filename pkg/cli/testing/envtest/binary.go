package envtest

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

const (
	// defaultK8sVersion is the default Kubernetes version for envtest binaries
	defaultK8sVersion = "1.28.0"
	// envtestBinEnvVar is the environment variable for custom binary location
	envtestBinEnvVar = "KUBEBUILDER_ASSETS"
)

// BinaryManager handles envtest binary download and caching
type BinaryManager struct {
	cacheDir   string
	k8sVersion string
}

// NewBinaryManager creates a new BinaryManager
func NewBinaryManager(k8sVersion string) (*BinaryManager, error) {
	if k8sVersion == "" {
		k8sVersion = defaultK8sVersion
	}

	cacheDir, err := getCacheDir()
	if err != nil {
		return nil, err
	}

	return &BinaryManager{
		cacheDir:   cacheDir,
		k8sVersion: k8sVersion,
	}, nil
}

// EnsureBinaries ensures envtest binaries are downloaded and cached
func EnsureBinaries() (string, error) {
	// First check if user has set custom binary location
	if customDir := os.Getenv(envtestBinEnvVar); customDir != "" {
		if binaryExists(customDir) {
			return customDir, nil
		}
	}

	// Get cache directory
	cacheDir, err := getCacheDir()
	if err != nil {
		return "", err
	}

	// Check if binaries already exist in cache
	binaryDir := filepath.Join(cacheDir, defaultK8sVersion, getPlatformDir())
	if binaryExists(binaryDir) {
		return binaryDir, nil
	}

	// Binaries don't exist - need to download
	// For now, return error with instructions
	return "", fmt.Errorf(
		"envtest binaries not found. Please run:\n"+
			"  go install sigs.k8s.io/controller-runtime/tools/setup-envtest@latest\n"+
			"  setup-envtest use %s --bin-dir %s\n"+
			"Or set %s environment variable to the binary directory",
		defaultK8sVersion, cacheDir, envtestBinEnvVar,
	)
}

// GetBinaryDir returns the directory containing envtest binaries
func (m *BinaryManager) GetBinaryDir() (string, error) {
	return EnsureBinaries()
}

// getCacheDir returns the cache directory for envtest binaries
func getCacheDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	cacheDir := filepath.Join(homeDir, ".kyverno", "envtest")
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return "", err
	}

	return cacheDir, nil
}

// getPlatformDir returns the platform-specific directory name
func getPlatformDir() string {
	return fmt.Sprintf("%s-%s", runtime.GOOS, runtime.GOARCH)
}

// binaryExists checks if all required binaries exist in the directory
func binaryExists(dir string) bool {
	requiredBinaries := []string{"kube-apiserver", "etcd"}

	// On Windows, binaries have .exe extension
	ext := ""
	if runtime.GOOS == "windows" {
		ext = ".exe"
	}

	for _, binary := range requiredBinaries {
		path := filepath.Join(dir, binary+ext)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			return false
		}
	}
	return true
}

// GetK8sVersion returns the Kubernetes version
func (m *BinaryManager) GetK8sVersion() string {
	return m.k8sVersion
}

// SetK8sVersion sets the Kubernetes version
func (m *BinaryManager) SetK8sVersion(version string) {
	m.k8sVersion = version
}
