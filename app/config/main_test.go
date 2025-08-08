package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidateWithValidConfig(t *testing.T) {
	// Create a temporary config file
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yml")
	
	validConfig := `mount_point: "/tmp/test"
volume_name: "test"
file_servers:
  - name: "test_server"
    target: "localhost:9090"
`
	
	if err := os.WriteFile(configPath, []byte(validConfig), 0644); err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}
	
	// Change to temp directory to test config loading
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer os.Chdir(oldWd)
	
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}
	
	if err := Validate(); err != nil {
		t.Errorf("Expected valid config to pass validation, got error: %v", err)
	}
}

func TestValidateWithMissingMountPoint(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yml")
	
	invalidConfig := `volume_name: "test"
file_servers:
  - name: "test_server"
    target: "localhost:9090"
`
	
	if err := os.WriteFile(configPath, []byte(invalidConfig), 0644); err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}
	
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer os.Chdir(oldWd)
	
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}
	
	err = Validate()
	if err == nil {
		t.Error("Expected validation to fail for missing mount_point")
	}
	if err.Error() != "mount_point is required" {
		t.Errorf("Expected specific error message, got: %v", err)
	}
}

func TestValidateWithMissingVolumeName(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yml")
	
	invalidConfig := `mount_point: "/tmp/test"
file_servers:
  - name: "test_server"
    target: "localhost:9090"
`
	
	if err := os.WriteFile(configPath, []byte(invalidConfig), 0644); err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}
	
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer os.Chdir(oldWd)
	
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}
	
	err = Validate()
	if err == nil {
		t.Error("Expected validation to fail for missing volume_name")
	}
	if err.Error() != "volume_name is required" {
		t.Errorf("Expected specific error message, got: %v", err)
	}
}

func TestValidateWithMissingFileServers(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yml")
	
	invalidConfig := `mount_point: "/tmp/test"
volume_name: "test"
file_servers: []
`
	
	if err := os.WriteFile(configPath, []byte(invalidConfig), 0644); err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}
	
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer os.Chdir(oldWd)
	
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}
	
	err = Validate()
	if err == nil {
		t.Error("Expected validation to fail for empty file_servers")
	}
	if err.Error() != "file_servers is required" {
		t.Errorf("Expected specific error message, got: %v", err)
	}
}

func TestGetMountPoint(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yml")
	
	expectedMountPoint := "/tmp/test_mount"
	validConfig := `mount_point: "` + expectedMountPoint + `"
volume_name: "test"
file_servers:
  - name: "test_server"
    target: "localhost:9090"
`
	
	if err := os.WriteFile(configPath, []byte(validConfig), 0644); err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}
	
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer os.Chdir(oldWd)
	
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}
	
	mountPoint, err := GetMountPoint()
	if err != nil {
		t.Fatalf("Expected GetMountPoint to succeed, got error: %v", err)
	}
	
	if mountPoint != expectedMountPoint {
		t.Errorf("Expected mount point %s, got %s", expectedMountPoint, mountPoint)
	}
}

func TestGetVolumeName(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yml")
	
	expectedVolumeName := "test_volume"
	validConfig := `mount_point: "/tmp/test"
volume_name: "` + expectedVolumeName + `"
file_servers:
  - name: "test_server"
    target: "localhost:9090"
`
	
	if err := os.WriteFile(configPath, []byte(validConfig), 0644); err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}
	
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer os.Chdir(oldWd)
	
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}
	
	volumeName, err := GetVolumeName()
	if err != nil {
		t.Fatalf("Expected GetVolumeName to succeed, got error: %v", err)
	}
	
	if volumeName != expectedVolumeName {
		t.Errorf("Expected volume name %s, got %s", expectedVolumeName, volumeName)
	}
}

func TestGetFileServers(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yml")
	
	validConfig := `mount_point: "/tmp/test"
volume_name: "test"
file_servers:
  - name: "server1"
    target: "localhost:9090"
  - name: "server2"
    target: "localhost:9091"
`
	
	if err := os.WriteFile(configPath, []byte(validConfig), 0644); err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}
	
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer os.Chdir(oldWd)
	
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}
	
	fileServers, err := GetFileServers()
	if err != nil {
		t.Fatalf("Expected GetFileServers to succeed, got error: %v", err)
	}
	
	expectedServers := []FileSystemProvider{
		{Name: "server1", Target: "localhost:9090"},
		{Name: "server2", Target: "localhost:9091"},
	}
	
	if len(fileServers) != len(expectedServers) {
		t.Errorf("Expected %d file servers, got %d", len(expectedServers), len(fileServers))
	}
	
	for i, expected := range expectedServers {
		if i >= len(fileServers) {
			t.Errorf("Missing file server at index %d", i)
			continue
		}
		
		actual := fileServers[i]
		if actual.Name != expected.Name {
			t.Errorf("Expected server name %s, got %s", expected.Name, actual.Name)
		}
		if actual.Target != expected.Target {
			t.Errorf("Expected server target %s, got %s", expected.Target, actual.Target)
		}
	}
}

func TestGetWithMissingConfigFile(t *testing.T) {
	tempDir := t.TempDir()
	
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer os.Chdir(oldWd)
	
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}
	
	_, err = get()
	if err == nil {
		t.Error("Expected get() to fail when config file is missing")
	}
}