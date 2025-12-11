package healthcheck

import (
	"fmt"
	"os"
	"path/filepath"
)

// HealthChecker provides methods to check the health of the mounted filesystem
type HealthChecker struct {
	mountpoint string
}

// New creates a new HealthChecker instance
func New(mountpoint string) *HealthChecker {
	return &HealthChecker{
		mountpoint: mountpoint,
	}
}

// GetHealthCheckFilePath returns the full path to the health check file
func (hc *HealthChecker) GetHealthCheckFilePath() string {
	return filepath.Join(hc.mountpoint, HealthCheckFileName)
}

// Check verifies that the mount is intact and readable by attempting to read the health check file
// Returns nil if healthy, or an error describing the issue
func (hc *HealthChecker) Check() error {
	filePath := hc.GetHealthCheckFilePath()

	// Check if file exists
	if _, err := os.Stat(filePath); err != nil {
		return fmt.Errorf("health check file not found at %s: %w", filePath, err)
	}

	// Attempt to read the file to verify the mount is functional
	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read health check file: %w", err)
	}

	// Verify the content is correct
	if string(content) != HealthCheckContent {
		return fmt.Errorf("health check file has invalid content: expected %q, got %q", HealthCheckContent, string(content))
	}

	return nil
}
