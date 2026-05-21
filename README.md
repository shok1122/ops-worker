# ops-worker

ops-worker is a lightweight machine state monitoring agent for Ubuntu, designed to run as a systemd service. It collects system metrics and sends them as JSON to a remote server.

## Features

- CPU, memory, disk usage monitoring
- Process and Docker container health checks
- External command integration
- Cron-based scheduling per check
- Periodic healthcheck reporting
- TLS support

## Installation

```bash
curl -fsSL https://raw.githubusercontent.com/shok1122/ops-worker/<tag>/install.sh | sudo bash -s <tag>
```

Example:

```bash
curl -fsSL https://raw.githubusercontent.com/shok1122/ops-worker/v1.0.0/install.sh | sudo bash -s v1.0.0
```

The script will:
1. Download and install the binary to `/usr/local/bin/ops-worker`
2. Create `/etc/ops-worker/` with sample config files (if not already present)
3. Install and start the systemd service

Edit `/etc/ops-worker/config.yaml` and `/etc/ops-worker/checks.yaml` to match your environment.

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
-config  path to config file (default: /etc/ops-worker/config.yaml)
-checks  path to checks file (overrides checks_file in config)
-version print version and exit
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
