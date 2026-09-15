package catalog

import (
	"fmt"
	"os"
	"path/filepath"
)

func LoadDirectory(dir string) (*Catalog, error) {
	catalog := NewCatalog()

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to read catalog directory %s: %w",
			dir,
			err,
		)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		ext := filepath.Ext(entry.Name())

		if ext != ".yaml" && ext != ".yml" {
			continue
		}

		path := filepath.Join(dir, entry.Name())

		fileCatalog, err := LoadFile(path)
		if err != nil {
			return nil, err
		}

		for _, diagnostics := range fileCatalog.Diagnostics {
			for _, diagnostic := range diagnostics {
				catalog.Add(diagnostic)
			}
		}
	}

	if err := Validate(catalog); err != nil {
		return nil, err
	}

	return catalog, nil
}
