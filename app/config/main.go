package config

import (
	"fmt"
	"os"

	"github.com/BurntSushi/toml"
)

type FileSystemProvider struct {
	Name   string `toml:"name"`
	Target string `toml:"target"`
}

type Config struct {
	MountPoint      string               `toml:"mount_point"`
	VolumeName      string               `toml:"volume_name"`
	Debug           bool                 `toml:"debug"`
	EnableDiskCache bool                 `toml:"enable_disk_cache"`
	FileServers     []FileSystemProvider `toml:"file_servers"`
}

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
