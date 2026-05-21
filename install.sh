#!/bin/bash
set -euo pipefail

REPO="shok1122/ops-worker"
INSTALL_DIR="/usr/local/bin"
CONFIG_DIR="/etc/ops-worker"
SERVICE_FILE="/etc/systemd/system/ops-worker.service"
BINARY_NAME="ops-worker"
RAW_BASE="https://raw.githubusercontent.com/${REPO}"

if [ $# -lt 1 ] || [ -z "$1" ]; then
  echo "Usage: $0 <tag>" >&2
  echo "Example: $0 v1.2.3" >&2
  exit 1
fi
TAG="$1"

# Get release binary URL from GitHub API
echo "Fetching release ${TAG}..."
LATEST_URL=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/tags/${TAG}" \
  | grep '"browser_download_url"' \
  | grep 'ops-worker"' \
  | cut -d'"' -f4)

if [ -z "$LATEST_URL" ]; then
  echo "ERROR: Could not find binary in release ${TAG}" >&2
  exit 1
fi

# Download and install binary
echo "Downloading ops-worker from ${LATEST_URL}..."
curl -fsSL "$LATEST_URL" -o "${INSTALL_DIR}/${BINARY_NAME}"
chmod +x "${INSTALL_DIR}/${BINARY_NAME}"
echo "Installed binary to ${INSTALL_DIR}/${BINARY_NAME}"

# Create config directory and download sample configs if not present
mkdir -p "$CONFIG_DIR"

if [ ! -f "${CONFIG_DIR}/config.yaml" ]; then
  curl -fsSL "${RAW_BASE}/${TAG}/config.example.yaml" -o "${CONFIG_DIR}/config.yaml"
  echo "Downloaded config.example.yaml to ${CONFIG_DIR}/config.yaml"
else
  echo "Config already exists at ${CONFIG_DIR}/config.yaml, skipping"
fi

if [ ! -f "${CONFIG_DIR}/checks.yaml" ]; then
  curl -fsSL "${RAW_BASE}/${TAG}/checks.example.yaml" -o "${CONFIG_DIR}/checks.yaml"
  echo "Downloaded checks.example.yaml to ${CONFIG_DIR}/checks.yaml"
else
  echo "Checks config already exists at ${CONFIG_DIR}/checks.yaml, skipping"
fi

# Install systemd unit file
curl -fsSL "${RAW_BASE}/${TAG}/ops-worker.service" -o "$SERVICE_FILE"
echo "Installed systemd unit file to ${SERVICE_FILE}"

# Enable and start service
systemctl daemon-reload
systemctl enable ops-worker
systemctl start ops-worker

echo "ops-worker installed and started successfully"
echo "Check status with: systemctl status ops-worker"
echo "View logs with: journalctl -u ops-worker -f"
