# GateRelay Security Hardening Checklist

This document provides a comprehensive security hardening checklist for the GateRelay deployment.

## Pre-Deployment Hardening

### Operating System

- [ ] **OS is up to date**
  ```bash
  sudo apt update && sudo apt upgrade -y
  ```

- [ ] **Automatic security updates enabled**
  ```bash
  sudo apt install unattended-upgrades
  sudo dpkg-reconfigure -plow unattended-upgrades
  ```

- [ ] **Unnecessary services disabled**
  ```bash
  systemctl list-unit-files --state=enabled
  # Disable any unnecessary services
  ```

- [ ] **Kernel hardening via sysctl**
  ```bash
  # Add to /etc/sysctl.conf or /etc/sysctl.d/99-hardening.conf
  net.ipv4.conf.all.rp_filter=1
  net.ipv4.conf.default.rp_filter=1
  net.ipv4.icmp_echo_ignore_broadcasts=1
  net.ipv4.conf.all.accept_redirects=0
  net.ipv4.conf.default.accept_redirects=0
  net.ipv4.conf.all.secure_redirects=0
  net.ipv4.conf.all.send_redirects=0
  net.ipv4.conf.default.accept_source_route=0
  net.ipv6.conf.all.accept_redirects=0
  net.ipv6.conf.default.accept_redirects=0
  kernel.randomize_va_space=2

  # Apply changes
  sudo sysctl -p
  ```

## Network Hardening

### Firewall (ufw)

- [ ] **Firewall installed and enabled**
  ```bash
  sudo apt install ufw
  sudo ufw enable
  ```

- [ ] **Default policies set**
  ```bash
  sudo ufw default deny incoming
  sudo ufw default allow outgoing
  ```

- [ ] **Only required ports open**
  ```bash
  # SSH (restricted to admin network)
  sudo ufw allow from 192.168.1.0/24 to any port 22 proto tcp

  # GateRelay service port
  sudo ufw allow 4000/tcp

  # Verify
  sudo ufw status numbered
  ```

- [ ] **Rate limiting on SSH**
  ```bash
  sudo ufw limit 22/tcp
  ```

- [ ] **Firewall logging enabled**
  ```bash
  sudo ufw logging on
  ```

- [ ] **Verify no unexpected open ports**
  ```bash
  sudo ss -tlnp
  ```

## SSH Hardening

- [ ] **SSH key authentication configured**
  ```bash
  # On your workstation, generate key if needed
  ssh-keygen -t ed25519 -C "admin@gaterelay"

  # Copy to server
  ssh-copy-id admin@gaterelay-host

  # Verify key login works before proceeding
  ```

- [ ] **Password authentication disabled**
  ```bash
  # In /etc/ssh/sshd_config
  PasswordAuthentication no
  ```

- [ ] **Root login disabled**
  ```bash
  # In /etc/ssh/sshd_config
  PermitRootLogin no
  ```

- [ ] **Strong ciphers configured**
  ```bash
  # Run hardening script
  sudo ./deploy/scripts/harden-ssh.sh

  # Or manually add to /etc/ssh/sshd_config
  Ciphers chacha20-poly1305@openssh.com,aes256-gcm@openssh.com,aes128-gcm@openssh.com
  MACs hmac-sha2-512-etm@openssh.com,hmac-sha2-256-etm@openssh.com
  KexAlgorithms curve25519-sha256,diffie-hellman-group-exchange-sha256
  ```

- [ ] **SSH configuration validated**
  ```bash
  sudo sshd -t
  sudo systemctl restart sshd
  ```

- [ ] **Verify SSH works after changes** (do not log out until verified!)
  ```bash
  # From another terminal
  ssh admin@gaterelay-host
  ```

## User and Access Control

### Service User

- [ ] **Dedicated service user created**
  ```bash
  grep gaterelay /etc/passwd
  # Should show: gaterelay:x:...:...::/nonexistent:/usr/sbin/nologin
  ```

- [ ] **Service user has no shell**
  ```bash
  # Should be /usr/sbin/nologin
  getent passwd gaterelay | cut -d: -f7
  ```

- [ ] **Service user not in sudo group**
  ```bash
  groups gaterelay
  # Should only show: gaterelay
  ```

### Admin Users

- [ ] **Admin users in sudo group only**
  ```bash
  grep sudo /etc/group
  ```

- [ ] **No unnecessary user accounts**
  ```bash
  cat /etc/passwd
  # Review all users, remove any unnecessary
  ```

- [ ] **Sudo requires password**
  ```bash
  # Verify /etc/sudoers does NOT have NOPASSWD for admin users
  sudo visudo -c
  ```

## File System Hardening

### Binary and Configuration

- [ ] **Binary permissions correct**
  ```bash
  ls -l /usr/local/bin/gaterelay
  # Should be: -rwxr-xr-x root root
  ```

- [ ] **Configuration directory permissions**
  ```bash
  ls -ld /etc/gaterelay/
  # Should be: drwxr-xr-x root root
  ```

- [ ] **Configuration file permissions**
  ```bash
  ls -l /etc/gaterelay/config.toml
  # Should be: -rw-r--r-- root root
  ```

- [ ] **State directory owned by service user**
  ```bash
  ls -ld /var/lib/gaterelay/
  # Should be: drwxr-x--- gaterelay gaterelay
  ```

- [ ] **No world-writable files in service directories**
  ```bash
  find /etc/gaterelay /var/lib/gaterelay /usr/local/bin/gaterelay -perm -002 -ls
  # Should return nothing
  ```

### systemd Unit File

- [ ] **systemd unit file permissions**
  ```bash
  ls -l /etc/systemd/system/gaterelay.service
  # Should be: -rw-r--r-- root root
  ```

## systemd Service Hardening

Verify hardening directives are in place:

- [ ] **Service runs as dedicated user**
  ```bash
  grep "User=gaterelay" /etc/systemd/system/gaterelay.service
  grep "Group=gaterelay" /etc/systemd/system/gaterelay.service
  ```

- [ ] **NoNewPrivileges enabled**
  ```bash
  grep "NoNewPrivileges=true" /etc/systemd/system/gaterelay.service
  ```

- [ ] **File system protections enabled**
  ```bash
  grep "ProtectSystem=strict" /etc/systemd/system/gaterelay.service
  grep "ProtectHome=true" /etc/systemd/system/gaterelay.service
  grep "PrivateTmp=true" /etc/systemd/system/gaterelay.service
  ```

- [ ] **Capabilities dropped**
  ```bash
  grep "CapabilityBoundingSet=$" /etc/systemd/system/gaterelay.service
  grep "AmbientCapabilities=$" /etc/systemd/system/gaterelay.service
  ```

- [ ] **Network restrictions in place**
  ```bash
  grep "RestrictAddressFamilies=AF_INET AF_INET6" /etc/systemd/system/gaterelay.service
  ```

- [ ] **Syscall filtering enabled**
  ```bash
  grep "SystemCallFilter=@system-service" /etc/systemd/system/gaterelay.service
  ```

- [ ] **Analyze security score**
  ```bash
  systemd-analyze security gaterelay
  # Should show low exposure score
  ```

## Application Configuration Hardening

- [ ] **Listen address appropriately restricted**
  ```bash
  # If only local access needed, use 127.0.0.1
  # If public access needed, use 0.0.0.0 (with firewall)
  grep "listen_addr" /etc/gaterelay/config.toml
  ```

- [ ] **Connection limits set**
  ```bash
  grep "max_conns" /etc/gaterelay/config.toml
  # Should be reasonable for your use case
  ```

- [ ] **Timeouts configured**
  ```bash
  grep "idle_timeout_secs" /etc/gaterelay/config.toml
  grep "connect_timeout_secs" /etc/gaterelay/config.toml
  # Should not be 0 (disabled) in production
  ```

- [ ] **Log level appropriate**
  ```bash
  grep "log_level" /etc/gaterelay/config.toml
  # Should be "info" or "warn" in production (not "debug")
  ```

## Logging and Monitoring

- [ ] **Service logs to journald**
  ```bash
  sudo journalctl -u gaterelay -n 10
  # Should show JSON logs
  ```

- [ ] **Journal size limits configured**
  ```bash
  grep -E "SystemMaxUse|MaxRetentionSec" /etc/systemd/journald.conf
  ```

- [ ] **Log rotation working**
  ```bash
  sudo journalctl --disk-usage
  # Should be under configured limit
  ```

- [ ] **Monitoring command available**
  ```bash
  # Test log monitoring
  sudo journalctl -u gaterelay -f &
  # Press Ctrl+C to stop
  ```

## Service Reliability

- [ ] **Service enabled on boot**
  ```bash
  sudo systemctl is-enabled gaterelay
  # Should return: enabled
  ```

- [ ] **Auto-restart configured**
  ```bash
  grep "Restart=on-failure" /etc/systemd/system/gaterelay.service
  ```

- [ ] **Restart limits set**
  ```bash
  grep "StartLimitInterval" /etc/systemd/system/gaterelay.service
  grep "StartLimitBurst" /etc/systemd/system/gaterelay.service
  ```

- [ ] **Test auto-restart**
  ```bash
  # Get PID
  PID=$(systemctl show -p MainPID gaterelay | cut -d= -f2)

  # Kill process
  sudo kill -9 $PID

  # Wait and check status
  sleep 5
  sudo systemctl status gaterelay
  # Should show: active (running)
  ```

## Network Security Testing

- [ ] **Verify firewall blocks unauthorized ports**
  ```bash
  # From external host, try connecting to blocked port
  nc -zv gaterelay-host 9999
  # Should fail/timeout

  # Check firewall logs
  sudo grep UFW /var/log/syslog | tail -10
  ```

- [ ] **Verify GateRelay port is open**
  ```bash
  # From external host
  nc -zv gaterelay-host 4000
  # Should succeed
  ```

- [ ] **Verify SSH is restricted** (if configured)
  ```bash
  # From unauthorized network
  nc -zv gaterelay-host 22
  # Should fail if restricted to admin network
  ```

- [ ] **Port scan detection**
  ```bash
  # Run nmap from external host
  nmap -sS gaterelay-host

  # Check for blocks in firewall logs
  sudo grep UFW /var/log/syslog | grep BLOCK
  ```

## Backup and Recovery

- [ ] **Configuration backed up**
  ```bash
  sudo tar czf /root/gaterelay-backup-$(date +%F).tar.gz \
      /etc/gaterelay/ \
      /etc/systemd/system/gaterelay.service \
      /usr/local/bin/gaterelay

  ls -lh /root/gaterelay-backup-*
  ```

- [ ] **Backup schedule configured** (optional)
  ```bash
  # Example cron job
  echo '0 2 * * * root tar czf /root/gaterelay-backup-$(date +\%F).tar.gz /etc/gaterelay/ /etc/systemd/system/gaterelay.service /usr/local/bin/gaterelay' | sudo tee /etc/cron.d/gaterelay-backup
  ```

- [ ] **Recovery procedure tested**
  ```bash
  # Extract backup in test environment
  # Verify service starts and operates correctly
  ```

## Documentation

- [ ] **Deployment documented**
  - IP addresses and network layout recorded
  - Configuration choices documented
  - Firewall rules documented

- [ ] **Operations runbook accessible**
  - OPERATIONS.md reviewed
  - Failure scenarios understood
  - Recovery procedures tested

- [ ] **Architecture understood**
  - ARCHITECTURE.md reviewed
  - Threat model understood
  - Security controls documented

## Final Verification

Run through these final checks:

- [ ] **Service is running**
  ```bash
  sudo systemctl status gaterelay
  # Should show: active (running)
  ```

- [ ] **Logs show healthy operation**
  ```bash
  sudo journalctl -u gaterelay -n 50
  # Should show startup and connection logs, no errors
  ```

- [ ] **Connections work end-to-end**
  ```bash
  # From client, connect through relay to target
  nc gaterelay-host 4000
  # Should connect to target service
  ```

- [ ] **Security score acceptable**
  ```bash
  systemd-analyze security gaterelay
  # Review exposure level (should be low)
  ```

- [ ] **No critical issues in logs**
  ```bash
  sudo journalctl -u gaterelay --since today | grep -i error
  # Investigate any errors
  ```

- [ ] **Firewall status verified**
  ```bash
  sudo ufw status verbose
  # Review all rules
  ```

- [ ] **All documentation complete**
  - README.md
  - DEPLOYMENT.md
  - OPERATIONS.md
  - ARCHITECTURE.md
  - This checklist

## Post-Deployment

- [ ] **Scheduled security review** (monthly recommended)
  - Review logs for anomalies
  - Check for OS updates
  - Verify firewall rules still appropriate
  - Audit user accounts

- [ ] **Incident response plan**
  - Contact information documented
  - Escalation procedures defined
  - Runbook tested

- [ ] **Monitoring alerts** (if applicable)
  - Service down alerts
  - High error rate alerts
  - Unusual traffic patterns

## Hardening Score

Calculate your hardening compliance:

**Total items:** ~80
**Items completed:** _____ / 80
**Compliance percentage:** _____%

**Target:** 95%+ for production deployment

## References

- [CIS Benchmarks for Ubuntu/Debian](https://www.cisecurity.org/cis-benchmarks/)
- [systemd Security Documentation](https://www.freedesktop.org/software/systemd/man/systemd.exec.html)
- [NIST Security Guidelines](https://www.nist.gov/cybersecurity)
- [OWASP Secure Coding Practices](https://owasp.org/www-project-secure-coding-practices-quick-reference-guide/)

**Last Updated:** 2026-01
**Review Frequency:** Monthly or after any significant changes
