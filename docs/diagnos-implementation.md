# Diagnos — Implementation Status

> Companion file to `diagnose-design.md` (architecture/design — unchanged, no need to restate it here).
> This file is the "where we actually are in code" snapshot. When resuming in a new chat, load both files: design doc for the *why/what*, this file for the *what's built and verified*.

## Environment
- Go 1.26, module: `github.com/faizahmd2/diagnos`
- Docker installed but deliberately not used yet — running everything via `go run` until the pipeline is proven end to end
- Ansible installed via brew (local machine)
- Test target: AWS EC2 instance, Ubuntu, reachable via `ssh ubuntu@ssh.quickorder.in`
- node_exporter running on the EC2 instance, port 9100 — confirmed reachable via `curl http://localhost:9100/metrics` on the box itself
- Prometheus running on the EC2 instance, port 9090 (not yet queried from Go — that's next)

## Repo tree (current state)

```
diagnos/
├── cmd/
│   └── investigator/
│       ├── main.go          ✅ implemented
│       ├── debug.go         ✅ implemented (logs only — pipeline not wired in yet)
│       └── probe.go         ✅ implemented (temporary — testing scaffold for the executor layer)
│
├── internal/
│   ├── config/
│   │   └── config.go        ✅ implemented
│   ├── executor/
│   │   └── ansible.go       ✅ implemented, verified working end to end
│   ├── collector/           ⬜ not started (next step)
│   ├── normalize/           ⬜ not started
│   ├── compare/             ⬜ not started
│   ├── catalog/             ⬜ not started
│   ├── planner/             ⬜ not started
│   ├── aiclient/            ⬜ not started
│   └── report/              ⬜ not started
│
├── ansible/
│   ├── inventory/
│   │   └── hosts.yaml       ✅ implemented, verified (ping + playbook both pass)
│   └── playbooks/
│       └── collect.yaml     ✅ implemented, verified
│
├── configs/
│   ├── app.yaml              ✅ implemented
│   └── catalog/               ⬜ not started
│
├── deployments/docker/        ⬜ empty placeholder, deferred intentionally
├── docs/                      (diagnose-design.md lives here)
├── scripts/                   ⬜ empty placeholder
├── go.mod
└── go.sum
```

## What's implemented and verified

- **Ansible connectivity** — `ansible quickorder-prod -i ansible/inventory/hosts.yaml -m ping` returns `pong`.
- **Collect playbook** — runs an arbitrary read-only shell command on the target host via extra-vars, verified manually with `df -h`.
- **Go → Ansible executor round trip** — `probe` command runs a command through the executor and prints real output from the EC2 box (`df -h` output confirmed, and a `curl` against node_exporter's `:9100/metrics` confirmed reachable through the same path).
- **Bug found and fixed today:** passing `target_host` and `cmd` as two separate `-e "key=value"` flags to `ansible-playbook` was unreliable. Fix: marshal both into a single JSON object and pass it as one `-e '<json>'` argument. This is the version below — this is the one to carry forward, not the earlier one from the setup step.

## Code — as it stands right now

### `configs/app.yaml`
```yaml
app:
  name: diagnos
  log_level: info

ai:
  provider: external   # external | internal
  external:
    api_key_env: DIAGNOS_AI_API_KEY
    base_url: https://api.anthropic.com
  internal:
    base_url: ""

thresholds:
  z_score_affected: 2.5
  max_areas_per_run: 2
  max_enrichment_rounds: 1

depth:
  default: standard   # quick | standard | deep

exporters:
  node_exporter:
    port: 9100
```

### `internal/config/config.go`
```go
package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	App struct {
		Name     string `yaml:"name"`
		LogLevel string `yaml:"log_level"`
	} `yaml:"app"`

	AI struct {
		Provider string `yaml:"provider"`
		External struct {
			APIKeyEnv string `yaml:"api_key_env"`
			BaseURL   string `yaml:"base_url"`
		} `yaml:"external"`
		Internal struct {
			BaseURL string `yaml:"base_url"`
		} `yaml:"internal"`
	} `yaml:"ai"`

	Thresholds struct {
		ZScoreAffected      float64 `yaml:"z_score_affected"`
		MaxAreasPerRun      int     `yaml:"max_areas_per_run"`
		MaxEnrichmentRounds int     `yaml:"max_enrichment_rounds"`
	} `yaml:"thresholds"`

	Depth struct {
		Default string `yaml:"default"`
	} `yaml:"depth"`

	Exporters struct {
		NodeExporter struct {
			Port int `yaml:"port"`
		} `yaml:"node_exporter"`
	} `yaml:"exporters"`
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

	return &cfg, nil
}
```

### `cmd/investigator/main.go`
```go
package main

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/spf13/cobra"
)

var (
	cfgPath string
	logger  *slog.Logger
)

func main() {
	logger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	rootCmd := &cobra.Command{
		Use:   "diagnos",
		Short: "Diagnos — machine health investigation engine",
	}

	rootCmd.PersistentFlags().StringVar(&cfgPath, "config", "configs/app.yaml", "path to config file")
	rootCmd.AddCommand(newDebugCmd())
	rootCmd.AddCommand(newProbeCmd())

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
```

### `cmd/investigator/debug.go`
*(unchanged since foundation step — logs only, pipeline not wired in yet)*
```go
package main

import (
	"github.com/spf13/cobra"

	"github.com/faizahmd2/diagnos/internal/config"
)

func newDebugCmd() *cobra.Command {
	var hint string

	cmd := &cobra.Command{
		Use:   "debug <machine/IP>",
		Short: "Run an investigation against a target machine",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			target := args[0]

			cfg, err := config.Load(cfgPath)
			if err != nil {
				return err
			}

			logger.Info("debug triggered",
				"target", target,
				"hint", hint,
				"depth", cfg.Depth.Default,
			)

			// TODO: collector -> normalize -> compare -> planner -> aiclient -> report

			return nil
		},
	}

	cmd.Flags().StringVar(&hint, "hint", "", "known-issue hint, e.g. 'disk:95%'")

	return cmd
}
```

### `cmd/investigator/probe.go`
*(TEMPORARY — this is a testing scaffold for the executor layer only, not part of the real pipeline. Expect this to be removed or repurposed once `collector`/`catalog` land and `debug` starts calling the executor for real.)*
```go
package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/faizahmd2/diagnos/internal/executor"
)

func newProbeCmd() *cobra.Command {
	var cmdToRun string

	cmd := &cobra.Command{
		Use:   "probe <inventory-host-alias>",
		Short: "Temporary: run a read-only command via Ansible (foundation testing only)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			host := args[0]

			exec := executor.NewAnsibleExecutor("ansible/inventory/hosts.yaml", "ansible/playbooks/collect.yaml")

			out, err := exec.Run(host, cmdToRun)
			if err != nil {
				return err
			}

			fmt.Println(out)
			return nil
		},
	}

	cmd.Flags().StringVar(&cmdToRun, "cmd", "hostname", "command to run (read-only only)")

	return cmd
}
```

### `internal/executor/ansible.go`
*(FIXED VERSION — this is the one that works, carry this forward, not any earlier version)*
```go
package executor

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
)

type AnsibleExecutor struct {
	InventoryPath string
	PlaybookPath  string
}

func NewAnsibleExecutor(inventoryPath, playbookPath string) *AnsibleExecutor {
	return &AnsibleExecutor{
		InventoryPath: inventoryPath,
		PlaybookPath:  playbookPath,
	}
}

type ansibleJSONOutput struct {
	Plays []struct {
		Tasks []struct {
			Task struct {
				Name string `json:"name"`
			} `json:"task"`
			Hosts map[string]struct {
				Stdout string `json:"stdout"`
				RC     int    `json:"rc"`
				Failed bool   `json:"failed"`
			} `json:"hosts"`
		} `json:"tasks"`
	} `json:"plays"`
}

// Run executes a single read-only command on targetHost via the collect playbook.
func (a *AnsibleExecutor) Run(targetHost, cmd string) (string, error) {
	extraVars := map[string]string{
		"target_host": targetHost,
		"cmd":         cmd,
	}

	extraVarsJSON, err := json.Marshal(extraVars)
	if err != nil {
		return "", fmt.Errorf("failed to marshal ansible variables: %w", err)
	}

	args := []string{
		"-i", a.InventoryPath,
		a.PlaybookPath,
		"-e", string(extraVarsJSON),
	}

	command := exec.Command("ansible-playbook", args...)
	command.Env = append(os.Environ(), "ANSIBLE_STDOUT_CALLBACK=json")

	var stdout, stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr

	if err := command.Run(); err != nil {
		return "", fmt.Errorf(
			"ansible-playbook failed: %w (stderr: %s)",
			err,
			stderr.String(),
		)
	}

	var parsed ansibleJSONOutput
	if err := json.Unmarshal(stdout.Bytes(), &parsed); err != nil {
		return "", fmt.Errorf(
			"failed to parse ansible json output: %w",
			err,
		)
	}

	if len(parsed.Plays) == 0 || len(parsed.Plays[0].Tasks) == 0 {
		return "", fmt.Errorf("no task results returned by ansible")
	}

	lastTask := parsed.Plays[0].Tasks[len(parsed.Plays[0].Tasks)-1]

	result, ok := lastTask.Hosts[targetHost]
	if !ok {
		return "", fmt.Errorf("no result found for host %s", targetHost)
	}

	if result.Failed || result.RC != 0 {
		return "", fmt.Errorf(
			"command failed on %s (rc=%d): %s",
			targetHost,
			result.RC,
			result.Stdout,
		)
	}

	return result.Stdout, nil
}
```

### `ansible/inventory/hosts.yaml`
```yaml
all:
  hosts:
    quickorder-prod:
      ansible_host: ssh.quickorder.in
      ansible_user: ubuntu
```

### `ansible/playbooks/collect.yaml`
```yaml
---
- hosts: "{{ target_host }}"
  gather_facts: no
  tasks:
    - name: run diagnostic command
      shell: "{{ cmd }}"
      register: result
```

## Verified test commands (all passed today)
```bash
ansible quickorder-prod -i ansible/inventory/hosts.yaml -m ping

ansible-playbook -i ansible/inventory/hosts.yaml ansible/playbooks/collect.yaml -e "target_host=quickorder-prod" -e "cmd='df -h'"

go run ./cmd/investigator probe quickorder-prod --cmd "df -h"

go run ./cmd/investigator probe quickorder-prod --cmd "curl -s http://localhost:9100/metrics | head -5"
```

## Next step (tomorrow)

Build `internal/collector/metrics` — a Go client that queries Prometheus directly on `:9090` (not through Ansible; Prometheus is already network-reachable over HTTP) for a specific metric, starting with something simple like `node_filesystem_avail_bytes`. This is the first real piece of the `collector` interface described in the design doc, and it's independent of everything built today — the executor/Ansible layer and the Prometheus collector don't depend on each other, so this can be built and tested in isolation before either gets wired into `debug`.
