#!/bin/bash
#
# GateRelay Installation Script
# Installs GateRelay binary, creates user, sets up directories, and configures systemd
#
# Usage: sudo ./install.sh
#

set -euo pipefail

# Configuration
BINARY_NAME="gaterelay"
INSTALL_DIR="/usr/local/bin"
CONFIG_DIR="/etc/gaterelay"
STATE_DIR="/var/lib/gaterelay"
SERVICE_USER="gaterelay"
SERVICE_GROUP="gaterelay"

echo "=== GateRelay Installation Script ==="
echo ""

# Check if running as root
if [[ $EUID -ne 0 ]]; then
   echo "This script must be run as root (use sudo)"
   exit 1
fi

# Check if binary exists
if [[ ! -f "$BINARY_NAME" ]]; then
    echo "Error: $BINARY_NAME binary not found in current directory"
    echo "Please build the binary first: go build -o $BINARY_NAME"
    exit 1
fi

echo "[1/7] Creating service user and group..."
if ! id -u "$SERVICE_USER" >/dev/null 2>&1; then
    useradd --system --no-create-home --shell /usr/sbin/nologin "$SERVICE_USER"
    echo "  ✓ Created user: $SERVICE_USER"
else
    echo "  ✓ User already exists: $SERVICE_USER"
fi

echo ""
echo "[2/7] Installing binary..."
install -m 755 "$BINARY_NAME" "$INSTALL_DIR/$BINARY_NAME"
echo "  ✓ Installed: $INSTALL_DIR/$BINARY_NAME"

echo ""
echo "[3/7] Creating directories..."
mkdir -p "$CONFIG_DIR"
chmod 755 "$CONFIG_DIR"
echo "  ✓ Created: $CONFIG_DIR"

mkdir -p "$STATE_DIR"
chown "$SERVICE_USER:$SERVICE_GROUP" "$STATE_DIR"
chmod 750 "$STATE_DIR"
echo "  ✓ Created: $STATE_DIR"

echo ""
echo "[4/7] Installing configuration..."
if [[ -f "$CONFIG_DIR/config.toml" ]]; then
    echo "  ! Config already exists: $CONFIG_DIR/config.toml"
    echo "    Backup created: $CONFIG_DIR/config.toml.backup.$(date +%s)"
    cp "$CONFIG_DIR/config.toml" "$CONFIG_DIR/config.toml.backup.$(date +%s)"
fi

if [[ -f "deploy/config/config.toml" ]]; then
    install -m 644 deploy/config/config.toml "$CONFIG_DIR/config.toml"
    echo "  ✓ Installed: $CONFIG_DIR/config.toml"
else
    echo "  ! Warning: deploy/config/config.toml not found"
    echo "    You will need to create $CONFIG_DIR/config.toml manually"
fi

echo ""
echo "[5/7] Installing systemd service..."
if [[ -f "deploy/systemd/gaterelay.service" ]]; then
    install -m 644 deploy/systemd/gaterelay.service /etc/systemd/system/gaterelay.service
    echo "  ✓ Installed: /etc/systemd/system/gaterelay.service"
else
    echo "  ! Error: deploy/systemd/gaterelay.service not found"
    exit 1
fi

echo ""
echo "[6/7] Reloading systemd..."
systemctl daemon-reload
echo "  ✓ Systemd reloaded"

echo ""
echo "[7/7] Verification..."
$INSTALL_DIR/$BINARY_NAME -version
echo ""

echo "=== Installation Complete ==="
echo ""
echo "Next steps:"
echo "  1. Edit configuration: $CONFIG_DIR/config.toml"
echo "  2. Start service:      sudo systemctl start gaterelay"
echo "  3. Enable on boot:     sudo systemctl enable gaterelay"
echo "  4. Check status:       sudo systemctl status gaterelay"
echo "  5. View logs:          sudo journalctl -u gaterelay -f"
echo ""
echo "Security hardening:"
echo "  - Run: sudo ./deploy/scripts/setup-firewall.sh"
echo "  - Run: sudo ./deploy/scripts/harden-ssh.sh"
echo ""
