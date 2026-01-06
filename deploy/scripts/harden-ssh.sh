#!/bin/bash
#
# SSH Hardening Script
# Applies security best practices to SSH configuration
#
# Usage: sudo ./harden-ssh.sh
#

set -euo pipefail

SSHD_CONFIG="/etc/ssh/sshd_config"
BACKUP_FILE="$SSHD_CONFIG.backup.$(date +%s)"

echo "=== SSH Hardening Script ==="
echo ""

# Check if running as root
if [[ $EUID -ne 0 ]]; then
   echo "This script must be run as root (use sudo)"
   exit 1
fi

echo "[1/3] Backing up SSH configuration..."
cp "$SSHD_CONFIG" "$BACKUP_FILE"
echo "  [OK] Backup created: $BACKUP_FILE"

echo ""
echo "[2/3] Applying hardening settings..."

# Function to set or update SSH config parameter
set_ssh_param() {
    local param="$1"
    local value="$2"

    if grep -q "^#*${param}" "$SSHD_CONFIG"; then
        sed -i "s/^#*${param}.*/${param} ${value}/" "$SSHD_CONFIG"
    else
        echo "${param} ${value}" >> "$SSHD_CONFIG"
    fi
    echo "  [OK] Set: $param $value"
}

# Disable root login
set_ssh_param "PermitRootLogin" "no"

# Disable password authentication (key-only)
set_ssh_param "PasswordAuthentication" "no"
set_ssh_param "PubkeyAuthentication" "yes"

# Disable empty passwords
set_ssh_param "PermitEmptyPasswords" "no"

# Disable challenge-response auth
set_ssh_param "ChallengeResponseAuthentication" "no"

# Disable X11 forwarding
set_ssh_param "X11Forwarding" "no"

# Use protocol 2 only (if not already default)
set_ssh_param "Protocol" "2"

# Set login grace time
set_ssh_param "LoginGraceTime" "60"

# Maximum authentication attempts
set_ssh_param "MaxAuthTries" "3"

# Maximum sessions
set_ssh_param "MaxSessions" "5"

# Use strong ciphers and MACs
cat >> "$SSHD_CONFIG" << 'EOF'

# Strong crypto settings (added by hardening script)
Ciphers chacha20-poly1305@openssh.com,aes256-gcm@openssh.com,aes128-gcm@openssh.com,aes256-ctr,aes192-ctr,aes128-ctr
MACs hmac-sha2-512-etm@openssh.com,hmac-sha2-256-etm@openssh.com,hmac-sha2-512,hmac-sha2-256
KexAlgorithms curve25519-sha256,curve25519-sha256@libssh.org,diffie-hellman-group-exchange-sha256
EOF

echo ""
echo "[3/3] Validating and restarting SSH..."

# Test configuration
if sshd -t; then
    echo "  [OK] Configuration valid"
    systemctl restart ssh
    echo "  [OK] SSH service restarted"
else
    echo "  [ERROR] Configuration invalid, restoring backup"
    cp "$BACKUP_FILE" "$SSHD_CONFIG"
    exit 1
fi

echo ""
echo "=== SSH Hardening Complete ==="
echo ""
echo "Applied settings:"
echo "  * Root login: DISABLED"
echo "  * Password auth: DISABLED (key-only)"
echo "  * Strong crypto: ENABLED"
echo ""
echo "IMPORTANT:"
echo "  Make sure you have SSH key access configured BEFORE logging out!"
echo "  If you get locked out, use VM console access to restore from: $BACKUP_FILE"
echo ""
