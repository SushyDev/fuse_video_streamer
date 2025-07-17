package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type FileSystemProvider struct {
	Name   string `yaml:"name"`
	Target string `yaml:"target"`
}

type Config struct {
	MountPoint  string               `yaml:"mount_point"`
	VolumeName  string               `yaml:"volume_name"`
	FileServers []FileSystemProvider `yaml:"file_servers"`
}

func get() (Config, error) {
	file, err := os.Open("config.yml")
	if err != nil {
		return Config{}, err
	}
	defer file.Close()

	decoder := yaml.NewDecoder(file)
	var cfg Config
	if err := decoder.Decode(&cfg); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

func Validate() error {
	cfg, err := get()
	if err != nil {
		return err
	}

	if cfg.MountPoint == "" {
		return fmt.Errorf("mount_point is required")
	}

	if cfg.VolumeName == "" {
		return fmt.Errorf("volume_name is required")
	}

	if len(cfg.FileServers) == 0 {
		return fmt.Errorf("file_servers is required")
	}

	return nil
}

func GetMountPoint() (string, error) {
	cfg, err := get()
	if err != nil {
		return "", err
	}
	return cfg.MountPoint, nil
}

func GetVolumeName() (string, error) {
	cfg, err := get()
	if err != nil {
		return "", err
	}
	return cfg.VolumeName, nil
}

func GetFileServers() ([]FileSystemProvider, error) {
	cfg, err := get()
	if err != nil {
		return nil, err
	}
	return cfg.FileServers, nil
}
