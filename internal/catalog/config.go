package catalog

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type fileCatalog struct {
	Area        string           `yaml:"area"`
	Diagnostics []fileDiagnostic `yaml:"diagnostics"`
}

type fileDiagnostic struct {
	ID             string       `yaml:"id"`
	Area           string       `yaml:"area"`
	Description    string       `yaml:"description"`
	Cost           string       `yaml:"cost"`
	Commands       []string     `yaml:"commands"`
	TimeoutSeconds int          `yaml:"timeout_seconds"`
	Trigger        *fileTrigger `yaml:"trigger"`
	LogSources     []string     `yaml:"log_sources"`
}

type fileTrigger struct {
	Signal   string  `yaml:"signal"`
	Operator string  `yaml:"operator"`
	Value    float64 `yaml:"value"`
}

func LoadFile(path string) (*Catalog, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to read catalog %s: %w",
			path,
			err,
		)
	}

	var raw fileCatalog

	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf(
			"failed to parse catalog %s: %w",
			path,
			err,
		)
	}

	catalog := NewCatalog()

	for _, item := range raw.Diagnostics {
		diagnostic := Diagnostic{
			ID:             item.ID,
			Area:           Area(raw.Area),
			Description:    item.Description,
			Cost:           Cost(item.Cost),
			Commands:       item.Commands,
			TimeoutSeconds: item.TimeoutSeconds,
			LogSources:     item.LogSources,
		}

		if item.Trigger != nil {
			diagnostic.Trigger = &Trigger{
				Signal:   item.Trigger.Signal,
				Operator: item.Trigger.Operator,
				Value:    item.Trigger.Value,
			}
		}

		catalog.Add(diagnostic)
	}

	if err := Validate(catalog); err != nil {
		return nil, fmt.Errorf(
			"invalid catalog %s: %w",
			path,
			err,
		)
	}

	return catalog, nil
}
