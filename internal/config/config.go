// 内部使用：internal/config/config.go
package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server struct {
		Host         string `yaml:"host"`
		Port         int    `yaml:"port"`
		ReadTimeout  int    `yaml:"read_timeout"`
		WriteTimeout int    `yaml:"write_timeout"`
	} `yaml:"server"`

	Database struct {
		Primary string `yaml:"primary"`
		L2      string `yaml:"l2"`
	} `yaml:"database"`

	Storage struct {
		VFSPath string `yaml:"vfs_path"`
	} `yaml:"storage"`
}

func LoadConfig(path string) (*Config, error) {
	buf, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := yaml.Unmarshal(buf, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
