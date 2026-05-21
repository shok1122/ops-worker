#!/bin/bash
set -euo pipefail

REPO="shok1122/ops-worker"
INSTALL_DIR="/usr/local/bin"
CONFIG_DIR="/etc/ops-worker"
SERVICE_FILE="/etc/systemd/system/ops-worker.service"
BINARY_NAME="ops-worker"

# Get latest release binary URL from GitHub API
echo "Fetching latest release..."
LATEST_URL=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" \
  | grep '"browser_download_url"' \
  | grep 'ops-worker"' \
  | cut -d'"' -f4)

if [ -z "$LATEST_URL" ]; then
  echo "ERROR: Could not find binary in latest release" >&2
  exit 1
fi

# Download and install binary
echo "Downloading ops-worker from ${LATEST_URL}..."
curl -fsSL "$LATEST_URL" -o "${INSTALL_DIR}/${BINARY_NAME}"
chmod +x "${INSTALL_DIR}/${BINARY_NAME}"
echo "Installed binary to ${INSTALL_DIR}/${BINARY_NAME}"

# Create config directory and copy sample configs if not present
mkdir -p "$CONFIG_DIR"

if [ ! -f "${CONFIG_DIR}/config.yaml" ]; then
  if [ -f "config.example.yaml" ]; then
    cp config.example.yaml "${CONFIG_DIR}/config.yaml"
    echo "Copied config.example.yaml to ${CONFIG_DIR}/config.yaml"
  else
    echo "WARN: config.example.yaml not found, skipping config installation" >&2
  fi
else
  echo "Config already exists at ${CONFIG_DIR}/config.yaml, skipping"
fi

if [ ! -f "${CONFIG_DIR}/checks.yaml" ]; then
  if [ -f "checks.example.yaml" ]; then
    cp checks.example.yaml "${CONFIG_DIR}/checks.yaml"
    echo "Copied checks.example.yaml to ${CONFIG_DIR}/checks.yaml"
  else
    echo "WARN: checks.example.yaml not found, skipping checks installation" >&2
  fi
else
  echo "Checks config already exists at ${CONFIG_DIR}/checks.yaml, skipping"
fi

# Install systemd unit file
if [ -f "ops-worker.service" ]; then
  cp ops-worker.service "$SERVICE_FILE"
  echo "Installed systemd unit file to ${SERVICE_FILE}"
else
  echo "WARN: ops-worker.service not found, skipping service installation" >&2
fi

# Enable and start service
systemctl daemon-reload
systemctl enable ops-worker
systemctl start ops-worker

echo "ops-worker installed and started successfully"
echo "Check status with: systemctl status ops-worker"
echo "View logs with: journalctl -u ops-worker -f"
