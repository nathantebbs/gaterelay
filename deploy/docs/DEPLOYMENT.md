# GateRelay Deployment Guide

Complete step-by-step guide for deploying GateRelay on Ubuntu Server LTS or Debian Stable.

## Prerequisites

- Fresh Ubuntu Server 22.04 LTS or Debian 12 (Stable) VM
- SSH access with sudo privileges
- Go 1.21+ installed (for building from source)
- Minimum 512MB RAM, 1 CPU core
- Network connectivity to target service

## Lab Network Layout Example

```
┌─────────────────────┐
│   Admin Workstation │
│   192.168.1.100     │
└──────────┬──────────┘
           │
           │ SSH (port 22)
           │
┌──────────▼──────────┐
│   GateRelay Host    │
│   192.168.1.10      │
│                     │
│   Listen: :4000     │
│   Target: 10.0.0.20 │
└──────────┬──────────┘
           │
           │ Relay (port 5000)
           │
┌──────────▼──────────┐
│   Target Service    │
│   10.0.0.20:5000    │
└─────────────────────┘
```

## Deployment Steps

### 1. Prepare the Build Environment

Clone or copy the GateRelay repository to your build machine:

```bash
git clone <repo-url>
cd gate-relay
```

Build the binary:

```bash
go build -o gaterelay
```

Verify the build:

```bash
./gaterelay -version
```

### 2. Transfer Files to Target Host

Transfer the necessary files to your deployment VM:

```bash
# Create tarball
tar czf gaterelay-deploy.tar.gz gaterelay deploy/

# Transfer to VM (adjust IP and user)
scp gaterelay-deploy.tar.gz admin@192.168.1.10:/tmp/

# SSH to VM
ssh admin@192.168.1.10
```

On the VM, extract the files:

```bash
cd /tmp
tar xzf gaterelay-deploy.tar.gz
cd gaterelay-deploy
```

### 3. Run Installation Script

The installation script will:
- Create the `gaterelay` system user
- Install the binary to `/usr/local/bin`
- Set up configuration in `/etc/gaterelay`
- Create state directory `/var/lib/gaterelay`
- Install systemd service unit

```bash
sudo ./deploy/scripts/install.sh
```

### 4. Configure the Service

Edit the configuration file:

```bash
sudo nano /etc/gaterelay/config.toml
```

Update the following parameters for your environment:

```toml
listen_addr = "0.0.0.0"        # Listen on all interfaces
listen_port = 4000             # Your ingress port

target_addr = "10.0.0.20"      # Your target service IP
target_port = 5000             # Your target service port

max_conns = 200                # Adjust based on expected load
idle_timeout_secs = 60         # Connection idle timeout
connect_timeout_secs = 5       # Target connection timeout
shutdown_grace_secs = 10       # Graceful shutdown timeout
log_level = "info"             # debug, info, warn, error
```

Verify configuration:

```bash
sudo /usr/local/bin/gaterelay -config /etc/gaterelay/config.toml &
# If it starts without error, kill it:
sudo pkill gaterelay
```

### 5. Configure Firewall

Run the firewall setup script:

```bash
# Basic setup (SSH from anywhere, GateRelay port open)
sudo ./deploy/scripts/setup-firewall.sh 4000 22 0.0.0.0/0

# Recommended: Restrict SSH to admin network
sudo ./deploy/scripts/setup-firewall.sh 4000 22 192.168.1.0/24
```

Verify firewall status:

```bash
sudo ufw status verbose
```

### 6. Harden SSH Access

**CRITICAL**: Ensure you have SSH key authentication set up before running this!

```bash
# Set up SSH key if not already done
mkdir -p ~/.ssh
chmod 700 ~/.ssh
# Add your public key to ~/.ssh/authorized_keys

# Run hardening script
sudo ./deploy/scripts/harden-ssh.sh
```

This will:
- Disable root login
- Disable password authentication (key-only)
- Configure strong crypto
- Rate limit authentication attempts

### 7. Start and Enable Service

Start the service:

```bash
sudo systemctl start gaterelay
```

Check status:

```bash
sudo systemctl status gaterelay
```

Enable on boot:

```bash
sudo systemctl enable gaterelay
```

View logs:

```bash
sudo journalctl -u gaterelay -f
```

### 8. Verify Operation

Test connectivity to the relay:

```bash
# From another machine
nc -zv 192.168.1.10 4000
```

Check logs for connection activity:

```bash
sudo journalctl -u gaterelay -n 50
```

## Post-Deployment Checklist

- [ ] Service starts successfully
- [ ] Logs show "relay started" message
- [ ] Firewall allows only required ports
- [ ] SSH is key-only (no password auth)
- [ ] Service restarts automatically after crash
- [ ] Test connection through relay reaches target
- [ ] Graceful shutdown works (sudo systemctl stop gaterelay)
- [ ] Service starts on system reboot

## Security Verification

Verify least privilege configuration:

```bash
# Check service user has no shell
grep gaterelay /etc/passwd
# Should show: gaterelay:x:...:...::/nonexistent:/usr/sbin/nologin

# Check file permissions
ls -la /usr/local/bin/gaterelay        # Should be root:root 755
ls -la /etc/gaterelay/                 # Should be root:root 755
ls -la /etc/gaterelay/config.toml      # Should be root:root 644
ls -la /var/lib/gaterelay/             # Should be gaterelay:gaterelay 750
```

Verify systemd hardening:

```bash
systemd-analyze security gaterelay
```

This should show a low "exposure level" score.

## Troubleshooting

### Service won't start

```bash
# Check logs
sudo journalctl -u gaterelay -n 100

# Test configuration
sudo /usr/local/bin/gaterelay -config /etc/gaterelay/config.toml

# Check permissions
sudo -u gaterelay /usr/local/bin/gaterelay -config /etc/gaterelay/config.toml
```

### Can't connect to relay port

```bash
# Check if service is listening
sudo ss -tlnp | grep 4000

# Check firewall
sudo ufw status verbose

# Test locally first
nc -zv localhost 4000
```

### Target connection failures

```bash
# Check target is reachable
ping 10.0.0.20
nc -zv 10.0.0.20 5000

# Check logs for specific error
sudo journalctl -u gaterelay | grep "failed to connect to target"
```

## Next Steps

- See [OPERATIONS.md](OPERATIONS.md) for daily operations and failure scenarios
- See [ARCHITECTURE.md](ARCHITECTURE.md) for system architecture details
