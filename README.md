## Diagnos

Diagnos is a single Go binary for collecting read-only machine evidence and
asking an LLM to produce an investigation report. It uses native SSH.

```sh
# A release contains only these two runtime files.
curl -LO https://example.invalid/diagnos
curl -LO https://example.invalid/app.yaml
chmod +x diagnos
# Edit app.yaml, then:
./diagnos config check
./diagnos debug production-vm --hint "intermittent latency"
```

`debug` accepts either a configured target alias or a host directly. SSH uses
the agent first, then an explicitly configured private key, then normal
`~/.ssh/id_*` keys. New hosts are prompted for by default; CI can use
`host_key_policy: strict` or `accept-new`.

Use `./diagnos debug my-host --dry-run` to inspect the embedded diagnostic
catalog without opening an SSH connection.

`app.yaml` is the production configuration. Its default
`output.report_type: app-metrics` writes a compact direct report and makes no
AI calls. Set it to `with-ai` only when the full AI investigation is wanted.

For a release build, run `make release VERSION=v1.0.0`. The `dist/` directory
contains only `diagnos` and `app.yaml`; the binary discovers the adjacent
configuration even when launched from another working directory.
