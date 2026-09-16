# Diagnos

Read-only Linux machine investigation CLI using native SSH, Prometheus, and optional LLM analysis.

## Install

Download the latest release:

[Diagnos Releases](https://github.com/faizahmd2/diagnos/releases/latest)

### macOS Apple Silicon

```sh
sudo mkdir -p /opt/diagnos

sudo curl -L \
  -o /opt/diagnos/diagnos \
  https://github.com/faizahmd2/diagnos/releases/latest/download/diagnos_v0.1.0_darwin_arm64

sudo curl -L \
  -o /opt/diagnos/app.yaml \
  https://github.com/faizahmd2/diagnos/releases/latest/download/app.yaml

sudo chmod +x /opt/diagnos/diagnos

sudo ln -sf /opt/diagnos/diagnos /usr/local/bin/diagnos
```

### Linux x86-64

```sh
sudo mkdir -p /opt/diagnos

sudo curl -L \
  -o /opt/diagnos/diagnos \
  https://github.com/faizahmd2/diagnos/releases/latest/download/diagnos_v0.1.0_linux_amd64

sudo curl -L \
  -o /opt/diagnos/app.yaml \
  https://github.com/faizahmd2/diagnos/releases/latest/download/app.yaml

sudo chmod +x /opt/diagnos/diagnos

sudo ln -sf /opt/diagnos/diagnos /usr/local/bin/diagnos
```

### Linux ARM64

```sh
sudo mkdir -p /opt/diagnos

sudo curl -L \
  -o /opt/diagnos/diagnos \
  https://github.com/faizahmd2/diagnos/releases/latest/download/diagnos_v0.1.0_linux_arm64

sudo curl -L \
  -o /opt/diagnos/app.yaml \
  https://github.com/faizahmd2/diagnos/releases/latest/download/app.yaml

sudo chmod +x /opt/diagnos/diagnos

sudo ln -sf /opt/diagnos/diagnos /usr/local/bin/diagnos
```

Check:

```sh
diagnos --version
```

## Configure

Edit the configuration:

```sh
sudo nano /opt/diagnos/app.yaml
```

Configure your target:

```yaml
targets:
  prod-vm-alias:
    host: 10.0.0.10
    user: ubuntu
    port: 22
```

The target name (`prod-vm-alias`) is used when running Diagnos.

Configure SSH if needed:

```yaml
ssh:
  host_key_policy: prompt
```

Available policies:

```text
prompt
strict
accept-new
insecure
```

Prometheus is optional.

AI is optional. For AI reports:

```yaml
output:
  report_type: with-ai
```

For direct reports:

```yaml
output:
  report_type: app-metrics
```

Add secrets to `app.yaml` or export with key.

## Validate

```sh
diagnos config check
```

## Run

Run from anywhere:

```sh
diagnos debug production-vm
```

Add an investigation hint:

```sh
diagnos debug production-vm \
  --hint "intermittent latency"
```

## Reports

Investigation output is written under the Diagnos installation directory:

```text
/opt/diagnos/debug/
```

For example:

```text
/opt/diagnos/debug/
├── diagnos-run.txt
├── context.txt
└── final-report.md
```

## Release Assets

Each release contains:

```text
diagnos_v0.1.0_darwin_arm64
diagnos_v0.1.0_linux_amd64
diagnos_v0.1.0_linux_arm64
app.yaml
checksums.txt
```

## License

See [LICENSE](LICENSE).
