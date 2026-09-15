package config

import (
	"fmt"
	"os"

	"github.com/faizahmd2/diagnos/internal/aiclient"
	"gopkg.in/yaml.v3"
)

type ApplicationLogConfig struct {
	Name    string   `yaml:"name"`
	Service string   `yaml:"service"`
	Paths   []string `yaml:"paths"`
}

type TargetConfig struct {
	Host string `yaml:"host"`
	User string `yaml:"user"`
}

type Config struct {
	App struct {
		Name     string `yaml:"name"`
		LogLevel string `yaml:"log_level"`
	} `yaml:"app"`

	Targets map[string]TargetConfig `yaml:"targets"`

	AI struct {
		Model  string                          `yaml:"model"`
		Models map[string]aiclient.ModelConfig `yaml:"models"`

		RequestLimits struct {
			MaxLinesContextFile int `yaml:"max_lines_context_file"`
		} `yaml:"request_limits"`
	} `yaml:"ai"`

	Telemetry struct {
		Prometheus struct {
			URL string `yaml:"url"`

			Auth struct {
				Type        string `yaml:"type"`
				TokenEnv    string `yaml:"token_env"`
				Username    string `yaml:"username"`
				PasswordEnv string `yaml:"password_env"`
			} `yaml:"auth"`
		} `yaml:"prometheus"`
	} `yaml:"telemetry"`

	Logs struct {
		Applications []ApplicationLogConfig `yaml:"applications"`
	} `yaml:"logs"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	if err := Validate(&cfg); err != nil {
		return nil, fmt.Errorf("validate config: %w", err)
	}

	return &cfg, nil
}
