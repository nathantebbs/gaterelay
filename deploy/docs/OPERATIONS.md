# GateRelay Operations Runbook

This document provides procedures for operating, monitoring, and troubleshooting the GateRelay service.

## Table of Contents

1. [Daily Operations](#daily-operations)
2. [Monitoring and Health Checks](#monitoring-and-health-checks)
3. [Configuration Changes](#configuration-changes)
4. [Log Management](#log-management)
5. [Failure Scenarios and Recovery](#failure-scenarios-and-recovery)
6. [Maintenance Procedures](#maintenance-procedures)

## Daily Operations

### Start Service

```bash
sudo systemctl start gaterelay
```

Expected log output:
```json
{"time":"2024-01-15T10:30:00Z","level":"INFO","msg":"starting gaterelay","version":"1.0.0","config":"/etc/gaterelay/config.toml"}
{"time":"2024-01-15T10:30:00Z","level":"INFO","msg":"relay started","listen_addr":"0.0.0.0:4000","target_addr":"10.0.0.20:5000","max_conns":200}
```

### Stop Service

Graceful shutdown:
```bash
sudo systemctl stop gaterelay
```

The service will:
1. Stop accepting new connections
2. Wait for active connections to close (up to `shutdown_grace_secs`)
3. Force close remaining connections after timeout

### Restart Service

```bash
sudo systemctl restart gaterelay
```

### Check Service Status

```bash
sudo systemctl status gaterelay
```

Expected output when healthy:
```
● gaterelay.service - GateRelay - Secure TCP Relay Service
     Loaded: loaded (/etc/systemd/system/gaterelay.service; enabled; vendor preset: enabled)
     Active: active (running) since ...
       Main PID: 12345 (gaterelay)
          Tasks: 8
         Memory: 5.2M
```

### Enable/Disable Auto-Start

```bash
# Enable on boot
sudo systemctl enable gaterelay

# Disable on boot
sudo systemctl disable gaterelay
```

## Monitoring and Health Checks

### View Real-Time Logs

```bash
# Follow all logs
sudo journalctl -u gaterelay -f

# Follow with 100 previous lines
sudo journalctl -u gaterelay -n 100 -f

# Filter by log level (ERROR only)
sudo journalctl -u gaterelay | grep '"level":"ERROR"'
```

### Check Connection Activity

```bash
# View recent connection logs
sudo journalctl -u gaterelay -n 50 | grep -E "(connection accepted|connection closed)"

# Count total connections today
sudo journalctl -u gaterelay --since today | grep -c "connection accepted"
```

### Monitor Active Connections

```bash
# Check listening port
sudo ss -tlnp | grep 4000

# See active connections to relay port
sudo ss -tn | grep :4000

# Count active connections
sudo ss -tn | grep :4000 | wc -l
```

### Resource Usage

```bash
# Memory and CPU usage
systemctl status gaterelay

# Detailed process info
ps aux | grep gaterelay

# Network statistics
sudo ss -s
```

### Log Indicators of Health

**Healthy logs:**
```json
{"level":"INFO","msg":"relay started"}
{"level":"INFO","msg":"connection accepted","conn_id":1,"client_addr":"192.168.1.50:54321"}
{"level":"INFO","msg":"connected to target","target_addr":"10.0.0.20:5000"}
{"level":"INFO","msg":"connection closed","bytes_rx":1024,"bytes_tx":2048}
```

**Warning indicators:**
```json
{"level":"WARN","msg":"connection limit reached","active_conns":200,"max_conns":200}
{"level":"ERROR","msg":"failed to connect to target","target_addr":"10.0.0.20:5000","error":"connection refused"}
```

## Configuration Changes

### Apply Configuration Changes

1. Edit configuration file:
```bash
sudo nano /etc/gaterelay/config.toml
```

2. Validate configuration:
```bash
sudo /usr/local/bin/gaterelay -config /etc/gaterelay/config.toml &
# If starts successfully:
sudo pkill gaterelay
```

3. Restart service:
```bash
sudo systemctl restart gaterelay
```

4. Verify:
```bash
sudo systemctl status gaterelay
sudo journalctl -u gaterelay -n 20
```

### Configuration Best Practices

- Always backup before changes:
  ```bash
  sudo cp /etc/gaterelay/config.toml /etc/gaterelay/config.toml.backup.$(date +%s)
  ```

- Test in staging environment first if possible
- Document changes in a change log
- Monitor logs closely after applying changes

## Log Management

### View Logs by Time Range

```bash
# Today's logs
sudo journalctl -u gaterelay --since today

# Last hour
sudo journalctl -u gaterelay --since "1 hour ago"

# Specific date range
sudo journalctl -u gaterelay --since "2024-01-15 00:00:00" --until "2024-01-15 23:59:59"
```

### Export Logs

```bash
# Export to file
sudo journalctl -u gaterelay --since today > gaterelay-logs-$(date +%F).log

# Export in JSON format
sudo journalctl -u gaterelay -o json --since today > gaterelay-logs-$(date +%F).json
```

### Log Rotation

journald automatically manages log rotation. Configuration is in `/etc/systemd/journald.conf`:

```ini
[Journal]
SystemMaxUse=500M
SystemKeepFree=1G
MaxRetentionSec=1month
```

Check current disk usage:
```bash
sudo journalctl --disk-usage
```

Manually clean old logs:
```bash
# Keep only last 7 days
sudo journalctl --vacuum-time=7d

# Keep only 100MB
sudo journalctl --vacuum-size=100M
```

## Failure Scenarios and Recovery

### Scenario 1: Target Service Down

**Symptom:**
```json
{"level":"ERROR","msg":"failed to connect to target","target_addr":"10.0.0.20:5000","error":"connection refused"}
```

**Impact:**
- New client connections fail to relay
- GateRelay continues running and accepting connections
- Each connection attempt logs an error

**Detection:**
```bash
sudo journalctl -u gaterelay -n 50 | grep "failed to connect to target"
```

**Recovery:**
1. Verify target service status:
   ```bash
   nc -zv 10.0.0.20 5000
   ```

2. Check target service (if you manage it):
   ```bash
   ssh target-host
   sudo systemctl status target-service
   ```

3. Restart target service:
   ```bash
   sudo systemctl restart target-service
   ```

4. Verify GateRelay can now connect:
   ```bash
   # Test from external machine
   nc 192.168.1.10 4000
   ```

**Prevention:**
- Monitor target service health
- Consider implementing retry logic or circuit breaker
- Set up alerts on repeated connection failures

### Scenario 2: GateRelay Service Crash

**Symptom:**
```bash
$ sudo systemctl status gaterelay
● gaterelay.service - GateRelay - Secure TCP Relay Service
     Active: failed (Result: signal) since ...
```

**Impact:**
- Service is down
- No connections are accepted
- Active connections are dropped

**Detection:**
```bash
# Service not running
sudo systemctl is-active gaterelay
# Returns: failed or inactive

# Check for crash in logs
sudo journalctl -u gaterelay -n 100 | grep -E "(killed|signal|crash)"
```

**Automatic Recovery:**

The systemd unit is configured with `Restart=on-failure`, so the service automatically restarts.

Verify auto-restart:
```bash
sudo systemctl status gaterelay
# Should show: Active: active (running)

# Check restart history
sudo journalctl -u gaterelay | grep "Started GateRelay"
```

**Manual Recovery (if auto-restart fails):**

1. Check why it's failing:
   ```bash
   sudo journalctl -u gaterelay -n 100
   ```

2. Common issues:
   - Configuration error: Validate config file
   - Port already in use: Check with `sudo ss -tlnp | grep 4000`
   - Permission issue: Check file permissions

3. Restart manually:
   ```bash
   sudo systemctl restart gaterelay
   ```

**Simulate Crash (for testing):**
```bash
# Get PID
PID=$(systemctl show -p MainPID gaterelay | cut -d= -f2)

# Send kill signal
sudo kill -9 $PID

# Watch auto-restart
sudo journalctl -u gaterelay -f
```

**Evidence to Collect:**
```bash
# Capture logs around crash time
sudo journalctl -u gaterelay --since "5 minutes ago" > crash-logs-$(date +%s).log

# System resource state
dmesg | tail -50
free -h
df -h
```

### Scenario 3: Port Scan / Unauthorized Access Attempt

**Symptom:**
Multiple connection attempts from unexpected sources, possibly with immediate disconnect.

**Detection:**
```bash
# Check for multiple connections from same IP
sudo journalctl -u gaterelay --since "1 hour ago" | grep "connection accepted" | awk '{print $NF}' | sort | uniq -c | sort -rn

# Check firewall logs (if logging enabled)
sudo grep UFW /var/log/syslog | tail -20
```

**Example unauthorized access attempt:**
```json
{"level":"INFO","msg":"connection accepted","client_addr":"45.123.45.67:54321"}
{"level":"ERROR","msg":"failed to connect to target","error":"i/o timeout"}
{"level":"INFO","msg":"connection closed","bytes_rx":0,"bytes_tx":0}
```

**Firewall Blocks:**
```
Jan 15 14:23:45 hostname kernel: [UFW BLOCK] IN=eth0 SRC=45.123.45.67 DST=192.168.1.10 PROTO=TCP DPT=22
```

**Response:**

1. Identify suspicious IPs:
   ```bash
   sudo journalctl -u gaterelay --since today | grep "connection accepted" | awk -F'"' '{print $12}' | cut -d: -f1 | sort | uniq -c | sort -rn
   ```

2. Block specific IP (if malicious):
   ```bash
   sudo ufw deny from 45.123.45.67
   sudo ufw reload
   ```

3. Review firewall rules:
   ```bash
   sudo ufw status numbered
   ```

4. If under active attack, consider:
   - Rate limiting: `sudo ufw limit 4000/tcp`
   - IP whitelist: Only allow known client IPs
   - Move to non-standard port (requires config change)

**Prevention:**
- Use firewall IP restrictions
- Enable UFW logging: `sudo ufw logging on`
- Monitor for unusual traffic patterns
- Consider fail2ban for automated blocking

### Scenario 4: Connection Limit Reached

**Symptom:**
```json
{"level":"WARN","msg":"connection limit reached, rejecting","active_conns":200,"max_conns":200,"remote_addr":"192.168.1.50:54321"}
```

**Impact:**
- New connections are rejected
- Existing connections continue working
- Legitimate users may be denied service

**Detection:**
```bash
# Count active connections
sudo ss -tn | grep :4000 | wc -l

# Check for limit warnings
sudo journalctl -u gaterelay -n 100 | grep "connection limit"
```

**Recovery:**

Short-term:
1. Increase connection limit in config:
   ```bash
   sudo nano /etc/gaterelay/config.toml
   # Change: max_conns = 400
   ```

2. Restart service:
   ```bash
   sudo systemctl restart gaterelay
   ```

Long-term:
- Analyze why limit is being hit (legitimate load vs. attack)
- Optimize idle timeout to recycle connections faster
- Scale horizontally (add more relay instances behind load balancer)
- Investigate client connection pooling issues

### Scenario 5: Configuration Error Preventing Start

**Symptom:**
```bash
$ sudo systemctl start gaterelay
Job for gaterelay.service failed because the control process exited with error code.
```

**Detection:**
```bash
sudo journalctl -u gaterelay -n 50
```

**Example error:**
```json
{"level":"ERROR","msg":"Configuration error: invalid configuration: listen_port must be between 1 and 65535, got 99999"}
```

**Recovery:**

1. Restore previous working config:
   ```bash
   # List backups
   ls -lt /etc/gaterelay/*.backup.*

   # Restore
   sudo cp /etc/gaterelay/config.toml.backup.TIMESTAMP /etc/gaterelay/config.toml
   ```

2. Fix the configuration error:
   ```bash
   sudo nano /etc/gaterelay/config.toml
   ```

3. Validate before restarting:
   ```bash
   sudo /usr/local/bin/gaterelay -config /etc/gaterelay/config.toml
   # Press Ctrl+C to stop after verifying it starts
   ```

4. Start service:
   ```bash
   sudo systemctl start gaterelay
   ```

## Maintenance Procedures

### Update GateRelay Binary

1. Build new version:
   ```bash
   go build -o gaterelay
   ./gaterelay -version
   ```

2. Transfer to server:
   ```bash
   scp gaterelay admin@192.168.1.10:/tmp/
   ```

3. On server, backup and replace:
   ```bash
   sudo cp /usr/local/bin/gaterelay /usr/local/bin/gaterelay.backup.$(date +%s)
   sudo install -m 755 /tmp/gaterelay /usr/local/bin/gaterelay
   ```

4. Restart service:
   ```bash
   sudo systemctl restart gaterelay
   ```

5. Verify:
   ```bash
   /usr/local/bin/gaterelay -version
   sudo systemctl status gaterelay
   ```

### System Updates

```bash
# Update system packages
sudo apt update
sudo apt upgrade -y

# Reboot if kernel updated
sudo reboot

# Verify service starts after reboot
sudo systemctl status gaterelay
```

### Backup Configuration

```bash
# Manual backup
sudo tar czf /root/gaterelay-backup-$(date +%F).tar.gz \
    /etc/gaterelay/ \
    /etc/systemd/system/gaterelay.service \
    /usr/local/bin/gaterelay

# Automated daily backup (cron)
echo '0 2 * * * root tar czf /root/gaterelay-backup-$(date +\%F).tar.gz /etc/gaterelay/ /etc/systemd/system/gaterelay.service' | sudo tee /etc/cron.d/gaterelay-backup
```

### Restore from Backup

```bash
# Extract backup
cd /
sudo tar xzf /root/gaterelay-backup-YYYY-MM-DD.tar.gz

# Reload systemd
sudo systemctl daemon-reload

# Restart service
sudo systemctl restart gaterelay
```

## Quick Reference Commands

```bash
# Status
sudo systemctl status gaterelay
sudo ss -tlnp | grep 4000

# Logs
sudo journalctl -u gaterelay -f
sudo journalctl -u gaterelay --since "1 hour ago"

# Control
sudo systemctl start gaterelay
sudo systemctl stop gaterelay
sudo systemctl restart gaterelay

# Config
sudo nano /etc/gaterelay/config.toml
sudo /usr/local/bin/gaterelay -config /etc/gaterelay/config.toml

# Monitoring
sudo ss -tn | grep :4000 | wc -l
sudo journalctl -u gaterelay | grep -c "connection accepted"
```
