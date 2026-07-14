package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config is the parsed representation of config.yaml.
type Config struct {
	Database struct {
		URL string `yaml:"url"`
	} `yaml:"database"`
	JWT struct {
		PrivateKeyFile string `yaml:"private_key_file"`
		PublicKeyFile  string `yaml:"public_key_file"` // unused: the app does not verify tokens
	} `yaml:"jwt"`
	HTTP struct {
		Health string `yaml:"health"`
		Public string `yaml:"public"`
	} `yaml:"http"`
}

// Load reads and validates the config file at path.
func Load(path string) (*Config, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config %q: %w", path, err)
	}
	var cfg Config
	if err := yaml.Unmarshal(raw, &cfg); err != nil {
		return nil, fmt.Errorf("parse config %q: %w", path, err)
	}
	if err := cfg.validate(); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func (c *Config) validate() error {
	switch {
	case c.Database.URL == "":
		return fmt.Errorf("config: database.url is required")
	case c.HTTP.Public == "":
		return fmt.Errorf("config: http.public is required")
	case c.HTTP.Health == "":
		return fmt.Errorf("config: http.health is required")
	case c.JWT.PrivateKeyFile == "":
		return fmt.Errorf("config: jwt.private_key_file is required")
	}
	return nil
}
