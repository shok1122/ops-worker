# ops-worker

ops-worker is a lightweight machine state monitoring agent for Ubuntu, designed to run as a systemd service. It collects system metrics and sends them as JSON to a remote server.

## Features

- CPU, memory, disk usage monitoring
- Process and Docker container health checks
- External command integration
- Cron-based scheduling per check
- Periodic healthcheck reporting
- Initial check execution on startup
- TLS support

## Installation

```bash
curl -fsSL https://raw.githubusercontent.com/shok1122/ops-worker/main/install.sh | sudo bash -s <tag>
```

Example:

```bash
curl -fsSL https://raw.githubusercontent.com/shok1122/ops-worker/main/install.sh | sudo bash -s v1.0.0
```

The script will:
1. Download and install the binary to `/usr/local/bin/ops-worker`
2. Create `/etc/ops-worker/` with sample config files (if not already present)
3. Install and start the systemd service

Edit `/etc/ops-worker/config.yaml` and `/etc/ops-worker/checks.yaml` to match your environment.

## Usage

ops-worker has three modes:

```
ops-worker <check-type> [args]                  run a check and print result to stdout
ops-worker --send <check-type> [args]           run a check once and send result to server
ops-worker --service [flags]                    run as background service
```

### One-shot mode (no config required)

Run a check and print the result as JSON to stdout:

```bash
ops-worker cpu
ops-worker memory
ops-worker disk /var/log
ops-worker process nginx
ops-worker docker my-app
ops-worker external /usr/local/bin/my-check.sh
```

### Send mode

Run a check once and send the result to the server (requires config file):

```bash
ops-worker --send cpu
ops-worker --send --config /etc/ops-worker/config.yaml disk /var/log
```

### Service mode

Run as a background service with cron-scheduled checks (requires config file):

```bash
ops-worker --service
ops-worker --service --config /etc/ops-worker/config.yaml
ops-worker --service --config /etc/ops-worker/config.yaml --checks /etc/ops-worker/checks.yaml
```

On startup, all checks run immediately once before the cron schedule takes over.

## Configuration

### config.yaml

```yaml
server:
  host: "example.com"
  port: 8443
  tls: true
  password: "secret-password"
  timeout: 10          # HTTP timeout in seconds (default: 10)
healthcheck:
  schedule: "*/5 * * * *"
checks_file: "/etc/ops-worker/checks.yaml"  # default
```

### checks.yaml

Each check has a name, type, cron schedule, and type-specific options.

```yaml
checks:
  - name: cpu
    type: cpu
    schedule: "*/1 * * * *"
    options: {}

  - name: memory
    type: memory
    schedule: "*/1 * * * *"
    options: {}

  - name: disk-root
    type: disk
    schedule: "*/5 * * * *"
    options:
      path: "/"

  - name: nginx-process
    type: process
    schedule: "*/1 * * * *"
    options:
      process_name: "nginx"

  - name: my-container
    type: docker
    schedule: "*/1 * * * *"
    options:
      container_name: "my-app"

  - name: custom
    type: external
    schedule: "*/5 * * * *"
    options:
      command: "/usr/local/bin/check.sh"
      args: []
      timeout: 10
```

### Supported checker types

| Type | Description | Options |
|------|-------------|---------|
| `cpu` | CPU usage | - |
| `memory` | Memory usage | - |
| `disk` | Disk usage | `path` (default: `/`) |
| `process` | Process running check | `process_name` |
| `docker` | Docker container check | `container_name` |
| `external` | External command | `command`, `args`, `timeout` |

### External checker output format

The external command must output JSON to stdout:

```json
{
  "status": "ok",
  "message": "all good",
  "metrics": [{"name": "value", "value": 42.0, "unit": "count"}],
  "labels": {"key": "value"}
}
```

## CLI flags

```
--config PATH   path to config file (default: /etc/ops-worker/config.yaml)
--checks PATH   path to checks file (service mode only, overrides checks_file in config)
--send          run a check once and send result to server
--service       run as background service
--version       print version and exit
```

## Building

```bash
go build -o ops-worker .
```

## systemd

```bash
systemctl status ops-worker
systemctl restart ops-worker
journalctl -u ops-worker -f
```
