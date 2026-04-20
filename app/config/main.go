// Package config handles loading and validation of the application configuration
// from a TOML file.
package config

import (
	"fmt"
	"os"

	"github.com/BurntSushi/toml"
)

// FileSystemProvider represents a single remote filesystem server to connect to.
type FileSystemProvider struct {
	Name   string `toml:"name"`
	Target string `toml:"target"`
}

// Config holds the application configuration loaded from config.toml.
type Config struct {
	MountPoint      string               `toml:"mount_point"`
	VolumeName      string               `toml:"volume_name"`
	Debug           bool                 `toml:"debug"`
	EnableDiskCache bool                 `toml:"enable_disk_cache"`
	FileServers     []FileSystemProvider `toml:"file_servers"`
}

// Get reads and validates the configuration from config.toml in the current directory.
func Get() (*Config, error) {
	configData, err := os.ReadFile("config.toml")
	if err != nil {
		return nil, err
	}

	var config *Config = &Config{}
	_, err = toml.Decode(string(configData), &config)
	if err != nil {
		return nil, err
	}

	err = validate(*config)
	if err != nil {
		return nil, err
	}

	return config, nil
}

func validate(cfg Config) error {
	if cfg.MountPoint == "" {
		return fmt.Errorf("mount_point is required")
	}

	if cfg.VolumeName == "" {
		return fmt.Errorf("volume_name is required")
	}

	if len(cfg.FileServers) == 0 {
		return fmt.Errorf("at least one file server is required")
	}

	return nil
}
