package config

import "fmt"

func (c Config) ResolveTarget(name string) (TargetConfig, error) {
	target, ok := c.Targets[name]
	if !ok {
		return TargetConfig{}, fmt.Errorf(
			"target %q is not configured",
			name,
		)
	}

	if target.Host == "" {
		return TargetConfig{}, fmt.Errorf(
			"target %q has no host configured",
			name,
		)
	}

	if target.User == "" {
		return TargetConfig{}, fmt.Errorf(
			"target %q has no user configured",
			name,
		)
	}

	return target, nil
}
