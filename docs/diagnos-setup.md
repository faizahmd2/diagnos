# Investigation Engine — Repository Structure & Getting Started

## Repository structure

```
diagnose/
├── cmd/
│   └── investigator/
│       └── main.go              # CLI entrypoint — `investigator debug <machine/IP>`
│
├── internal/
│   ├── collector/
│   │   ├── interface.go         # common Collector interface (source-agnostic)
│   │   ├── metrics/             # Prometheus (or equivalent) client
│   │   └── logs/                # journalctl / Loki / app-log clients
│   │
│   ├── normalize/
│   │   ├── drain.go             # Drain (Go port) wrapper — log template mining
│   │   └── metrics_stats.go     # rolling baseline, mean/stddev/z-score/delta calc
│   │
│   ├── compare/
│   │   └── engine.go            # deterministic comparison: current vs baseline → evidence
│   │
│   ├── catalog/
│   │   └── ubuntu/
│   │       ├── disk.go          # predefined disk command group (+ fallback chain)
│   │       ├── cpu.go
│   │       ├── memory.go
│   │       ├── kernel.go
│   │       └── network.go
│   │
│   ├── executor/
│   │   └── ansible.go           # Ansible runner — enforces allowlist, read-only only
│   │
│   ├── planner/
│   │   └── planner.go           # decides affected areas, enrichment round cap, area fan-out cap
│   │
│   ├── aiclient/
│   │   ├── interface.go         # provider-agnostic AI client interface
│   │   ├── internal_api.go      # company-internal AI API implementation
│   │   └── external_api.go      # external provider (own API key) implementation
│   │
│   ├── report/
│   │   ├── schema.go            # response schema types (call 1 / call 2)
│   │   └── format.go            # renders final human-readable report
│   │
│   └── config/
│       └── config.go            # loads app.yaml, env overrides
│
├── configs/
│   ├── app.yaml                 # provider selection, thresholds, depth levels, round caps
│   └── catalog/
│       └── ubuntu.yaml          # command catalog definitions (source of truth, loaded by catalog/ubuntu/*.go)
│
├── ansible/
│   ├── inventory/
│   │   └── hosts.yaml           # target machines, ssh + read-only sudo config
│   └── playbooks/
│       └── collect.yaml         # runs catalog commands read-only, returns output
│
├── deployments/
│   └── docker/
│       ├── Dockerfile
│       └── docker-compose.yml   # investigator service (+ any local deps)
│
├── docs/
│   └── diagnose-design.md   # the design doc from earlier
│
├── scripts/
│   └── setup.sh                 # one-shot local bootstrap
│
├── go.mod
├── go.sum
└── README.md
```

**Why this shape:**
- `collector/` and `aiclient/` are both interfaces first — swapping Prometheus for something else, or the internal AI API for an external key, never touches `planner/` or `compare/`.
- `catalog/` is split by OS family folder (`ubuntu/` now) so adding a distro later means adding a folder, not touching existing logic.
- `executor/` is the single choke point that talks to Ansible — this is where the allowlist enforcement lives, so no other package can execute anything on a target machine.
- `normalize/` and `compare/` are pure, deterministic, no AI calls — testable without ever hitting a model.

## Getting started

1. **Prerequisites**
   - Go (latest stable)
   - Docker + Docker Compose
   - Ansible installed locally (the executor shells out to it)
   - SSH access to at least one target machine, with a read-only sudo user configured on that machine

2. **Clone and init**
   ```
   git clone <repo-url> diagnose
   cd diagnose
   go mod tidy
   ```

3. **Configure the AI provider**
   - Edit `configs/app.yaml` — set `ai.provider` to either `internal` or `external`.
   - For `external`, set the API key via environment variable (never commit it) — e.g. `export INVESTIGATOR_AI_API_KEY=...`
   - For `internal`, set the internal endpoint URL in the same file.

4. **Configure target machine access**
   - Add the target machine(s) to `ansible/inventory/hosts.yaml` with SSH details and the read-only sudo user.
   - Sanity check connectivity: `ansible all -i ansible/inventory/hosts.yaml -m ping`

5. **Review the command catalog**
   - `configs/catalog/ubuntu.yaml` holds the starting disk/CPU/memory/kernel/network command groups with fallback chains.
   - Nothing here should ever be a write command — review this file before pointing it at any real machine.

6. **Set thresholds and caps**
   - In `configs/app.yaml`: z-score threshold for "affected", max areas per run, max enrichment rounds per area, depth levels (quick/standard/deep).

7. **Build and run locally**
   ```
   go build -o bin/investigator ./cmd/investigator
   ./bin/investigator debug <machine-ip>
   ```
   With a known-issue hint:
   ```
   ./bin/investigator debug <machine-ip> --hint "disk:95%"
   ```

8. **Run via Docker (once collectors/AI client are wired up)**
   ```
   docker compose -f deployments/docker/docker-compose.yml up --build
   ```

Start by wiring up `collector/metrics` against a real Prometheus instance and `normalize/metrics_stats.go` first — that's the smallest end-to-end slice you can verify without touching Ansible or the AI client at all.
