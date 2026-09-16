package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

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
	Port int    `yaml:"port"`
}

type SSHConfig struct {
	User           string        `yaml:"user"`
	Port           int           `yaml:"port"`
	KeyPath        string        `yaml:"key_path"`
	KnownHosts     string        `yaml:"known_hosts"`
	HostKeyPolicy  string        `yaml:"host_key_policy"`
	JumpHosts      []string      `yaml:"jump_hosts"`
	ConnectTimeout time.Duration `yaml:"connect_timeout"`
	CommandTimeout time.Duration `yaml:"command_timeout"`
	MaxParallel    int           `yaml:"max_parallel"`
	MaxOutputBytes int64         `yaml:"max_output_bytes"`
}

type Config struct {
	SSH SSHConfig `yaml:"ssh"`
	App struct {
		Name     string `yaml:"name"`
		LogLevel string `yaml:"log_level"`
	} `yaml:"app"`

	Targets map[string]TargetConfig `yaml:"targets"`

	AI struct {
		Provider string                          `yaml:"provider"`
		APIKey   string                          `yaml:"api_key"`
		BaseURL  string                          `yaml:"base_url"`
		Timeout  time.Duration                   `yaml:"timeout"`
		Model    string                          `yaml:"model"`
		Models   map[string]aiclient.ModelConfig `yaml:"models"`

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

	Output struct {
		ReportType string `yaml:"report_type"`
	} `yaml:"output"`
}

func Load(path string) (*Config, error) {
	cfg := defaults()
	if path == "" {
		path = DiscoverPath()
	}
	if path != "" {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		if err := yaml.Unmarshal(data, &cfg); err != nil {
			return nil, err
		}
	}
	applyEnv(&cfg)

	if err := Validate(&cfg); err != nil {
		return nil, fmt.Errorf("validate config: %w", err)
	}

	return &cfg, nil
}

// DiscoverPath returns the first conventional config file. An empty result is
// intentional: a default config plus environment/SSH-agent authentication is valid.
func DiscoverPath() string {
	// app.yaml is the production configuration. configs/app.yaml remains a
	// compatibility location for existing installations.
	candidates := []string{}
	// A downloaded release consists of only diagnos and app.yaml. Prefer the
	// adjacent config so running it from another working directory still works.
	if executable, err := os.Executable(); err == nil {
		dir := filepath.Dir(executable)
		candidates = append(candidates, filepath.Join(dir, "app.yaml"), filepath.Join(dir, "app.yml"))
	}
	candidates = append(candidates, "app.yaml", "app.yml", filepath.Join("configs", "app.yaml"), filepath.Join("configs", "app.yml"))
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		candidates = append(candidates, filepath.Join(xdg, "diagnos", "config.yml"))
	} else if home, err := os.UserHomeDir(); err == nil {
		candidates = append(candidates, filepath.Join(home, ".config", "diagnos", "config.yml"), filepath.Join(home, ".diagnos", "config.yml"))
	}
	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	return ""
}
func defaults() Config {
	var cfg Config
	cfg.App.Name = "diagnos"
	cfg.App.LogLevel = "info"
	cfg.SSH.Port = 22
	cfg.SSH.HostKeyPolicy = "prompt"
	cfg.SSH.ConnectTimeout = 10 * time.Second
	cfg.SSH.CommandTimeout = 30 * time.Second
	cfg.SSH.MaxParallel = 4
	cfg.SSH.MaxOutputBytes = 1024 * 1024
	// The direct report is intentionally compact; this is the shared evidence
	// budget used by both report modes.
	cfg.AI.RequestLimits.MaxLinesContextFile = 250
	cfg.Output.ReportType = "app-metrics"
	cfg.AI.Model = "gemini/gemini-3.5-flash-lite"
	cfg.AI.Models = map[string]aiclient.ModelConfig{
		cfg.AI.Model: {BaseURL: "https://generativelanguage.googleapis.com", APIKeyEnv: "DIAGNOS_AI_API_KEY"},
	}
	return cfg
}
func applyEnv(cfg *Config) {
	if v := os.Getenv("DIAGNOS_SSH_PORT"); v != "" {
		fmt.Sscanf(v, "%d", &cfg.SSH.Port)
	}
	if v := os.Getenv("DIAGNOS_SSH_USER"); v != "" {
		cfg.SSH.User = v
	}
	if v := os.Getenv("DIAGNOS_SSH_KEY_PATH"); v != "" {
		cfg.SSH.KeyPath = v
	}
	if v := os.Getenv("DIAGNOS_AI_API_KEY"); v != "" { // existing model adapters use their named env var; retain the value for config check only.
		cfg.AI.APIKey = v
		if cfg.AI.Models == nil {
			cfg.AI.Models = map[string]aiclient.ModelConfig{}
		}
		if cfg.AI.Model != "" {
			model := cfg.AI.Models[cfg.AI.Model]
			if model.APIKeyEnv == "" {
				model.APIKeyEnv = "DIAGNOS_AI_API_KEY"
				cfg.AI.Models[cfg.AI.Model] = model
			}
		}
	}
	cfg.SSH.KeyPath = strings.TrimSpace(cfg.SSH.KeyPath)
}
