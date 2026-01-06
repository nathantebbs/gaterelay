#!/bin/bash
#
# GateRelay Firewall Configuration Script
# Configures ufw to allow only necessary ports
#
# Usage: sudo ./setup-firewall.sh [LISTEN_PORT] [SSH_PORT] [ADMIN_NETWORK]
#

set -euo pipefail

# Default values
LISTEN_PORT="${1:-4000}"
SSH_PORT="${2:-22}"
ADMIN_NETWORK="${3:-0.0.0.0/0}"  # Change to restrict SSH access

echo "=== GateRelay Firewall Setup ==="
echo "Listen Port: $LISTEN_PORT"
echo "SSH Port: $SSH_PORT"
echo "Admin Network: $ADMIN_NETWORK"
echo ""

# Check if running as root
if [[ $EUID -ne 0 ]]; then
   echo "This script must be run as root (use sudo)"
   exit 1
fi

# Install ufw if not present
if ! command -v ufw &> /dev/null; then
    echo "Installing ufw..."
    apt-get update
    apt-get install -y ufw
fi

echo "Configuring firewall rules..."

# Reset to defaults
ufw --force reset

# Set default policies
ufw default deny incoming
ufw default allow outgoing

# Allow SSH from admin network only
echo "Allowing SSH from $ADMIN_NETWORK on port $SSH_PORT"
ufw allow from "$ADMIN_NETWORK" to any port "$SSH_PORT" proto tcp comment 'SSH Admin Access'

# Allow GateRelay listen port from anywhere
echo "Allowing GateRelay on port $LISTEN_PORT"
ufw allow "$LISTEN_PORT"/tcp comment 'GateRelay Service'

# Rate limiting on SSH to prevent brute force
ufw limit "$SSH_PORT"/tcp comment 'SSH Rate Limit'

# Enable logging (can generate verbose logs, adjust as needed)
ufw logging on

# Enable firewall
echo ""
echo "Enabling firewall..."
ufw --force enable

# Show status
echo ""
echo "=== Firewall Status ==="
ufw status verbose

echo ""
echo "=== Firewall Configuration Complete ==="
echo ""
echo "IMPORTANT SECURITY NOTES:"
echo "1. SSH is allowed from: $ADMIN_NETWORK"
echo "2. To restrict SSH to specific IPs, re-run with: sudo $0 $LISTEN_PORT $SSH_PORT <IP/CIDR>"
echo "3. Example: sudo $0 4000 22 192.168.1.0/24"
echo "4. GateRelay service port $LISTEN_PORT is open to all (0.0.0.0/0)"
echo ""
