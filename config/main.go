package config

import (
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

func get() Config {
	file, err := os.Open("config.yml")
	if err != nil {
		panic(err)
	}
	defer file.Close()

	decoder := yaml.NewDecoder(file)
	var cfg Config
	if err := decoder.Decode(&cfg); err != nil {
		panic(err)
	}

	return cfg
}

func Validate() {
	cfg := get()

	if cfg.MountPoint == "" {
		panic("MountPoint is required")
	}

	if cfg.VolumeName == "" {
		panic("VolumeName is required")
	}

	if len(cfg.FileServers) == 0 {
		panic("FileServers is required")
	}
}

func GetMountPoint() string {
	cfg := get()
	return cfg.MountPoint
}

func GetVolumeName() string {
	cfg := get()
	return cfg.VolumeName
}

func GetFileServers() []FileSystemProvider {
	cfg := get()
	return cfg.FileServers
}
