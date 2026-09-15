package config

import (
	"fmt"
	"strings"
)

func Validate(cfg *Config) error {
	if cfg == nil {
		return fmt.Errorf("config is nil")
	}

	if len(cfg.Targets) == 0 {
		return fmt.Errorf("no targets configured")
	}

	for name, target := range cfg.Targets {
		if strings.TrimSpace(name) == "" {
			return fmt.Errorf("target name cannot be empty")
		}

		if strings.TrimSpace(target.Host) == "" {
			return fmt.Errorf("target %q has empty host", name)
		}

		if strings.TrimSpace(target.User) == "" {
			return fmt.Errorf("target %q has empty user", name)
		}
	}

	if strings.TrimSpace(cfg.Telemetry.Prometheus.URL) == "" {
		return fmt.Errorf("prometheus URL is required")
	}

	if strings.TrimSpace(cfg.AI.Model) == "" {
		return fmt.Errorf("AI model is required")
	}

	if _, ok := cfg.AI.Models[cfg.AI.Model]; !ok {
		return fmt.Errorf(
			"AI model %q is not configured under ai.models",
			cfg.AI.Model,
		)
	}

	if cfg.AI.RequestLimits.MaxLinesContextFile <= 0 {
		return fmt.Errorf("ai.request_limits.max_lines_context_file must be greater than 0")
	}

	return nil
}
