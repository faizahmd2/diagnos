package resources

import (
	"fmt"
	"os"
)

func WritePlaybook() (string, func(), error) {
	data, err := Files.ReadFile("ansible/collect.yaml")
	if err != nil {
		return "", nil, fmt.Errorf("read embedded ansible playbook: %w", err)
	}

	file, err := os.CreateTemp("", "diagnos-collect-*.yaml")
	if err != nil {
		return "", nil, fmt.Errorf("create temporary ansible playbook: %w", err)
	}

	cleanup := func() {
		_ = os.Remove(file.Name())
	}

	if _, err := file.Write(data); err != nil {
		_ = file.Close()
		cleanup()
		return "", nil, fmt.Errorf("write temporary ansible playbook: %w", err)
	}

	if err := file.Close(); err != nil {
		cleanup()
		return "", nil, fmt.Errorf("close temporary ansible playbook: %w", err)
	}

	return file.Name(), cleanup, nil
}
