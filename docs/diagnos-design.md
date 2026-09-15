# Machine Investigation Engine — Design Document

## 1. Problem statement
- Observability tools (Prometheus/Grafana, cloud logging, alerting) already solve "is my system healthy" and "alert me on threshold breach" — that part is not being rebuilt.
- What's missing: when an alert fires, a human still has to manually correlate metrics, dig through logs, run diagnostic commands, and reason about root cause. That process typically takes an engineer 1–2 hours of manual investigation.
- Goal: compress that 1–2 hour manual investigation into a 2–5 minute automated pass that produces a human-readable report — not a fix, not an autonomous agent, just the investigation work a senior engineer would do by hand.

## 2. Approach
- Not "dump all logs/metrics into an LLM prompt and ask for a diagnosis" — that's the dead-simple approach and produces low-quality, unfocused answers.
- Instead: deterministic collection and comparison first, LLM reasoning only over pre-filtered, compact evidence — two bounded calls, not an agentic loop.
- Call 1 (narrow down): give the model enough compact evidence to identify *which* subsystems are affected and how confident it is.
- Call 2+ (go deep): for each confirmed affected area, pull deeper targeted data for that specific area and ask for the full investigation report — carrying call 1's history plus the new area-specific context.
- No autonomous command execution by the model. No arbitrary shell strings from an LLM response. No writes, ever.

## 3. Current tools in market & the gap
- Prometheus + Grafana, Loki, cloud-native observability (CloudWatch, GCP Ops, Datadog, etc.) — solve collection, storage, visualization, and threshold-based alerting well. Actively improving their own AI layers too.
- Gap: none of them do the *investigation* step — correlating a metric spike with recent log patterns, process behavior, and disk/process state to produce a reasoned hypothesis with confidence and evidence trail. That correlation work is still manual.
- Our contribution is the intelligence/investigation layer on top of existing telemetry — not another metrics store, not another dashboard.

## 4. Operating considerations & access model
- User/service account has **read-only** access to target machines, with sudo scoped to read-only diagnostic commands only (e.g. `sudo systemctl status`, never `sudo systemctl restart`).
- Execution happens via Ansible against the target machine(s).
- Predefined command catalog, grouped by domain (disk, CPU, memory, kernel/system, network) — the model never invents a command string; it only selects from a known-safe enum, and the enum itself is what's allowlisted at the execution layer (defense in depth — even if a bad value slipped through, Ansible would reject anything off-list).
- Commands include fallback chains for tool availability differences, e.g.: try `systemctl status <svc>` → if it fails or the binary is unavailable, fall back to `service <svc> status` → if that's unavailable, fall back to reading `/proc` or `journalctl` directly for the same signal. This fallback logic lives in the command catalog definition, not decided at runtime by the LLM.
- All commands are strictly read-only — no writes, no config changes, no restarts, ever, at any depth.

## 5. Data sources & collectors
- **Metrics (TSDB):** Prometheus (or equivalent) for CPU, memory, disk, network time series — both current snapshot and historical window (last 15m/30m/1d configurable).
- **Logs:** journalctl, Loki, or any configured log backend — system logs, service logs, kernel logs.
- **Application-level logs:** API request/response logs where available, for services running on the box — treated as another log source through the same collector interface, not a special case.
- **Live diagnostic commands:** Ansible-executed, predefined, read-only (per section 4).
- All collectors implement a common interface so the source is swappable — Prometheus today, something else tomorrow — without changing the shape the rest of the pipeline consumes.

## 6. AI call layer
- Provider-agnostic by design: pluggable between a company's internal AI request API and a direct provider API (own API key) — swapping providers should not require touching the investigation logic, only the AI client implementation.
- Local models (Ollama) explicitly avoided for this project given hardware constraints (16GB RAM, largely consumed) and the tool-calling/context-heavy nature of the reasoning calls.
- Always exactly 2+ calls, bounded — never an open-ended agent loop. A fixed cap on enrichment rounds (e.g. max one deep-dive round per affected area, max N affected areas per run) prevents runaway calls.

## 7. Deployment model
- Entire application ships as dockerized microservices.
- No data leaves the boundary of the system running the investigation — collectors, comparison engine, and AI client all run inside the same containerized boundary; only the AI call itself goes external (to whichever provider is configured).

## 8. Capabilities, happy paths, trade-offs
**Happy path:**
- `debug <machine/IP>` (optionally with a known-issue hint, e.g. "disk at 95%") → collectors gather current + historical data → comparison engine flags deviations → call 1 identifies affected areas + confidence → call 2 produces a full report with evidence, investigation paths considered, findings, and suggested remediation steps (not applied).

**Capabilities:**
- Snapshot mode (current state only) and investigate mode (current vs. historical, root-cause style).
- Depth control (quick/standard/deep) governing how many domains and how much historical context get pulled in.
- Multi-area investigation with bounded fan-out.

**Trade-offs:**
- Read-only-only means the tool can diagnose but never remediate — by design, but worth stating plainly.
- Predefined command catalogs mean coverage is only as good as the catalog; a novel failure mode outside the catalog's signals won't be caught until the catalog is extended.
- Ubuntu-only at first — other distros will surface gaps in the command catalog that aren't yet handled.
- Bounded calls mean the system won't chase an issue indefinitely — an inconclusive result after the enrichment cap is a valid, expected outcome, not a failure state.

## 9. Implementation — stack & dependencies
- **Language:** Go (collector, comparison engine, orchestration service).
- **Log compression:** Drain (Go port) for log template mining — clusters repeated log lines into templates with count, first/last seen, and sample line, instead of raw dumps.
- **Metrics comparison:** plain statistical aggregation (mean/stddev/z-score/delta) against a rolling baseline window — no external library needed.
- **Machine access/execution:** Ansible, read-only sudo scope.
- **Metrics/log backends:** Prometheus (metrics), journalctl/Loki (logs) — pluggable via a common collector interface.
- **AI client:** provider-agnostic HTTP client, swappable between internal company API and external provider API key.
- **Packaging:** Docker / docker-compose for the microservices.

## 10. Configuration layers
- Ubuntu only for the first version — base command set (systemd, journalctl, standard `/proc` and disk tooling) is consistent across the distro.
- Command catalog is keyed by OS family so other distros (RHEL/CentOS, etc.) can be added later without touching the core investigation logic — deferred, not designed for yet.

## 11. Lifecycle — full flow after `debug` trigger
1. **Trigger:** `debug <machine/IP>` issued via CLI, optionally carrying a known-issue hint (e.g. disk 95% alert).
2. **Collection:** collectors pull current snapshot + historical window (metrics from TSDB, logs from journalctl/Loki/app logs) for the target machine.
3. **Normalization:** raw log output run through Drain to produce compact templates (count, first/last seen, sample line); metrics run through the statistical comparison layer to produce deltas/z-scores against baseline. All of this — logs, metrics, and command output — is appended and compacted into a single consolidated evidence artifact for the run, not sent as separate per-source payloads.
4. **Known-issue biasing:** if a hint was provided (e.g. disk 95%), it biases which channels the comparison engine flags before the AI ever sees the data.
5. **Call 1 (narrow down):** consolidated evidence artifact sent to the AI layer → response identifies affected area(s) with a confidence indicator per area, informed by the deterministic severity/z-score already computed (not invented by the model).
6. **Area-level decision:** for each affected area above the confidence threshold, decide whether more targeted data is needed (bounded to a max number of areas/rounds).
7. **Targeted collection:** for confirmed areas, run the predefined read-only command group for that domain via Ansible (with fallback chains where applicable) to gather deeper, area-specific evidence.
8. **Call 2 (report):** sent with call 1's history + the new area-specific evidence → returns the final structured report: signals observed, investigation paths considered, findings, and suggested steps (no fixes executed, no writes).
9. **Output:** structured report returned to the CLI / caller.

## 12. Standard response schema & re-operations
- **Call 1 response (per run):** list of affected areas, each with a confidence value and the evidence fields that justified it (traceable back to a specific source command/exporter — no unsupported claims allowed).
- **Call 2 response (per area, or combined if multiple areas were bundled):** signals observed, investigation paths/channels considered, findings per path, root-cause hypothesis with confidence, and suggested remediation steps (informational only, never auto-applied).
- **Re-operation semantics:** if confidence from call 1 is below threshold for an area, the system does not loop indefinitely — it either runs the single allowed enrichment round (call 2 scoped to that area) or, if still inconclusive after that, the final report marks that area as "inconclusive — needs manual investigation" rather than issuing further calls.
- Every value in every response must carry provenance back to the specific command, exporter, or log source it came from, so the report never states a cause that isn't traceable to actual collected evidence.
