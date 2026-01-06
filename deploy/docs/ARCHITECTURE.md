# GateRelay Architecture

Technical architecture documentation for the GateRelay secure TCP relay service.

## System Overview

GateRelay is a production-hardened TCP relay service designed to forward connections from a public ingress port to a controlled egress target. It demonstrates enterprise-grade systems administration practices including least privilege, defense in depth, and operational excellence.

### Design Principles

1. **Least Privilege**: Runs as unprivileged user with minimal capabilities
2. **Defense in Depth**: Multiple security layers (systemd, firewall, application)
3. **Fail Secure**: Errors result in connection rejection, not bypass
4. **Observable**: Structured logging for all significant events
5. **Resilient**: Automatic recovery from failures

## Component Architecture

```
+--------------------------------------------------------------+
|                        GateRelay Host                        |
|                                                              |
|  +-----------------------------------------------------------+|
|  |                    Firewall (ufw)                       | |
|  |  * Default deny incoming                                | |
|  |  * Allow SSH (restricted to admin network)              | |
|  |  * Allow :4000 (relay ingress)                          | |
|  |  * Rate limiting on SSH                                 | |
|  +-----------------------+-----------------------------------+ |
|                          |                                    |
|  +-----------------------v-----------------------------------+ |
|  |              systemd Hardening Layer                    | |
|  |  * NoNewPrivileges=true                                 | |
|  |  * ProtectSystem=strict                                 | |
|  |  * PrivateTmp=true                                      | |
|  |  * RestrictAddressFamilies=AF_INET AF_INET6             | |
|  |  * CapabilityBoundingSet= (empty)                       | |
|  |  * SystemCallFilter=@system-service                     | |
|  +-----------------------+-----------------------------------+ |
|                          |                                    |
|  +-----------------------v-----------------------------------+ |
|  |              GateRelay Application                      | |
|  |                                                         | |
|  |  +------------------------------------------------------+| |
|  |  |  Config Loader (TOML)                              | | |
|  |  |  * Validation                                      | | |
|  |  |  * Sensible defaults                               | | |
|  |  +------------------------------------------------------+| |
|  |                                                         | |
|  |  +------------------------------------------------------+| |
|  |  |  TCP Listener (Ingress)                            | | |
|  |  |  * Bind to listen_addr:listen_port                 | | |
|  |  |  * Accept loop with context cancellation           | | |
|  |  |  * Connection limit enforcement                    | | |
|  |  +------------------------------------------------------+| |
|  |                                                         | |
|  |  +------------------------------------------------------+| |
|  |  |  Connection Handler (per-connection goroutine)     | | |
|  |  |  * Client -> Target dialer with timeout            | | |
|  |  |  * Bidirectional io.Copy                           | | |
|  |  |  * Idle timeout management                         | | |
|  |  |  * Byte counter statistics                         | | |
|  |  +------------------------------------------------------+| |
|  |                                                         | |
|  |  +------------------------------------------------------+| |
|  |  |  Signal Handler & Graceful Shutdown                | | |
|  |  |  * SIGTERM/SIGINT handling                         | | |
|  |  |  * Stop accepting new connections                  | | |
|  |  |  * Wait for active connections (timeout)           | | |
|  |  |  * Statistics logging on exit                      | | |
|  |  +------------------------------------------------------+| |
|  |                                                         | |
|  |  +------------------------------------------------------+| |
|  |  |  Structured Logger (slog)                          | | |
|  |  |  * JSON output to stdout -> journald               | | |
|  |  |  * Contextual fields (conn_id, client_addr, etc.)  | | |
|  |  |  * Configurable log level                          | | |
|  |  +------------------------------------------------------+| |
|  +---------------------------------------------------------+ |
|                                                              |
+--------------------------------------------------------------+
```

## Data Flow

### Connection Lifecycle

```
1. Client Connection
   Client --TCP--> GateRelay:4000
                      |
                      +- Accept()
                      +- Check max_conns
                      +- Log "connection accepted"
                      +- Spawn handler goroutine

2. Target Connection
   GateRelay --Dial()--> Target:5000
                             |
                             +- Apply connect_timeout
                             +- On success: Log "connected to target"
                             +- On failure: Log error, close client

3. Bidirectional Relay
   Client <--io.Copy--> GateRelay <--io.Copy--> Target
             goroutine 1            goroutine 2
                             |
                             +- Count bytes_rx (client->target)
                             +- Count bytes_tx (target->client)
                             +- Apply idle_timeout on both sides

4. Connection Termination
   Either side closes/EOF/timeout
                      |
                      +- Close both connections
                      +- Update global counters
                      +- Log "connection closed" with stats
                      +- Decrement active_conns
```

### Shutdown Flow

```
SIGTERM/SIGINT received
         |
         +- Log "received shutdown signal"
         +- Cancel context (stops accept loop)
         +- Close listener (reject new connections)
         |
         +- Wait for all handlers to complete
         |   (with shutdown_grace_secs timeout)
         |
         +- If timeout: Log warning, force close
         |
         +- Log final statistics and exit
```

## Security Architecture

### Attack Surface Reduction

| Layer | Protection | Mitigation |
|-------|-----------|------------|
| **Network** | Firewall (ufw) | Only expose necessary ports; rate limit SSH |
| **Access** | SSH hardening | Key-only auth, no root login, strong crypto |
| **Process** | systemd sandboxing | No new privileges, capability dropping, syscall filtering |
| **Filesystem** | Permissions | Read-only binary, protected config, isolated state dir |
| **Application** | Input validation | Port ranges, timeout bounds, connection limits |

### Principle of Least Privilege

**Service User:**
- Username: `gaterelay`
- Shell: `/usr/sbin/nologin` (no interactive login)
- Home: none (or `/nonexistent`)
- Sudo: not in sudoers
- Group: `gaterelay` (dedicated)

**Filesystem Permissions:**
```
/usr/local/bin/gaterelay       root:root       755  (executable by all, writable by root)
/etc/gaterelay/                root:root       755  (readable by all)
/etc/gaterelay/config.toml     root:root       644  (readable by all, writable by root)
/var/lib/gaterelay/            gaterelay:gaterelay 750 (only gaterelay can read/write)
```

**systemd Capabilities:**
- `CapabilityBoundingSet=` (empty) - No Linux capabilities
- `AmbientCapabilities=` (empty) - No ambient capabilities
- `NoNewPrivileges=true` - Cannot gain privileges via setuid

### systemd Hardening Measures

| Directive | Purpose | Impact |
|-----------|---------|--------|
| `ProtectSystem=strict` | Read-only `/usr`, `/boot`, `/etc` | Cannot modify system files |
| `ProtectHome=true` | No access to `/home`, `/root` | Cannot read user data |
| `PrivateTmp=true` | Private `/tmp` namespace | Isolates temp files |
| `PrivateDevices=true` | No access to physical devices | Cannot access hardware |
| `RestrictAddressFamilies=AF_INET AF_INET6` | Only TCP/IP networking | Cannot use Unix sockets, etc. |
| `SystemCallFilter=@system-service` | Whitelist safe syscalls | Blocks dangerous operations |
| `MemoryDenyWriteExecute=true` | W^X memory pages | Prevents code injection |
| `LockPersonality=true` | No personality changes | Prevents kernel quirk abuse |

## File System Layout

```
/usr/local/bin/
+-- gaterelay                 # Service binary (built from Go)

/etc/gaterelay/
+-- config.toml              # Service configuration

/etc/systemd/system/
+-- gaterelay.service        # systemd unit file

/var/lib/gaterelay/          # State directory (currently unused, reserved for future)

/var/log/journal/            # journald logs (managed by systemd)
+-- system.journal

Deployment artifacts (not deployed, for reference):
./deploy/
+-- config/
|   +-- config.toml          # Example configuration
+-- systemd/
|   +-- gaterelay.service    # systemd unit file
+-- scripts/
|   +-- install.sh           # Installation automation
|   +-- setup-firewall.sh    # Firewall configuration
|   +-- harden-ssh.sh        # SSH hardening
+-- docs/
    +-- DEPLOYMENT.md        # Deployment guide
    +-- OPERATIONS.md        # Operations runbook
    +-- ARCHITECTURE.md      # This document
```

## Configuration Schema

```toml
# Network configuration
listen_addr = "0.0.0.0"           # Interface to bind (0.0.0.0 = all, 127.0.0.1 = localhost)
listen_port = 4000                # Ingress port (1-65535)
target_addr = "10.0.0.20"         # Target host IP/hostname
target_port = 5000                # Target port (1-65535)

# Resource limits
max_conns = 200                   # Maximum concurrent connections (min: 1)

# Timeouts (seconds, 0 = disabled)
idle_timeout_secs = 60            # Close idle connections
connect_timeout_secs = 5          # Target connection timeout
shutdown_grace_secs = 10          # Graceful shutdown timeout

# Logging
log_level = "info"                # debug, info, warn, error
```

**Validation:**
- Port ranges: 1-65535
- Timeouts: >= 0
- Max connections: >= 1
- Log level: must be valid level string
- Target address: cannot be empty

## Logging Architecture

### Log Format

GateRelay uses structured JSON logging via Go's `log/slog` package, writing to stdout which systemd-journald captures.

**Example log entries:**

Startup:
```json
{"time":"2024-01-15T10:30:00.123Z","level":"INFO","msg":"starting gaterelay","version":"1.0.0","config":"/etc/gaterelay/config.toml"}
{"time":"2024-01-15T10:30:00.145Z","level":"INFO","msg":"relay started","listen_addr":"0.0.0.0:4000","target_addr":"10.0.0.20:5000","max_conns":200}
```

Connection accepted:
```json
{"time":"2024-01-15T10:31:05.234Z","level":"INFO","msg":"connection accepted","conn_id":1,"client_addr":"192.168.1.50:54321"}
```

Target connected:
```json
{"time":"2024-01-15T10:31:05.256Z","level":"INFO","msg":"connected to target","conn_id":1,"client_addr":"192.168.1.50:54321","target_addr":"10.0.0.20:5000"}
```

Connection closed:
```json
{"time":"2024-01-15T10:32:15.789Z","level":"INFO","msg":"connection closed","conn_id":1,"client_addr":"192.168.1.50:54321","bytes_rx":4096,"bytes_tx":8192,"error":null}
```

Error (target unreachable):
```json
{"time":"2024-01-15T10:33:00.456Z","level":"ERROR","msg":"failed to connect to target","conn_id":2,"client_addr":"192.168.1.51:54322","target_addr":"10.0.0.20:5000","error":"dial tcp 10.0.0.20:5000: connect: connection refused"}
```

Shutdown:
```json
{"time":"2024-01-15T10:35:00.000Z","level":"INFO","msg":"received shutdown signal","signal":"terminated"}
{"time":"2024-01-15T10:35:10.123Z","level":"INFO","msg":"all connections closed gracefully"}
{"time":"2024-01-15T10:35:10.145Z","level":"INFO","msg":"shutdown complete","total_conns":47,"bytes_rx":1048576,"bytes_tx":2097152}
```

### Log Retention

Managed by systemd-journald (default: `/etc/systemd/journald.conf`):
- **Disk limit:** SystemMaxUse=500M (or system-dependent)
- **Retention:** MaxRetentionSec=1month
- **Location:** `/var/log/journal/`

Access logs via `journalctl`:
```bash
sudo journalctl -u gaterelay -f             # Follow
sudo journalctl -u gaterelay --since today  # Today's logs
sudo journalctl -u gaterelay -o json        # JSON output
```

## Performance Characteristics

### Resource Usage (Typical)

Based on a VM with 1 vCPU, 512MB RAM:

| Metric | Idle | 50 Connections | 200 Connections |
|--------|------|----------------|-----------------|
| **Memory** | ~5-8 MB | ~12 MB | ~20 MB |
| **CPU** | <1% | 2-5% | 5-15% |
| **File Descriptors** | 5 | ~105 | ~405 |

Notes:
- Each connection uses 2 file descriptors (client + target)
- Go runtime includes overhead (~5MB base)
- Goroutine overhead is minimal (~2KB per goroutine)

### Scalability

**Vertical scaling:**
- Max connections limited by:
  - File descriptor limit (see `/proc/sys/fs/file-max`)
  - Available memory
  - CPU for data copying
- Recommended: Test at expected load, tune `max_conns`

**Horizontal scaling:**
- Deploy multiple GateRelay instances on different ports or hosts
- Use external load balancer (HAProxy, nginx) to distribute
- Each instance is stateless (no shared state required)

### Throughput

Relay throughput depends on:
- Network bandwidth between relay and target
- Data size per connection
- Go's `io.Copy` efficiency (generally very good)

Typical: Saturates 1Gbps network with large transfers (100MB+ files)

## Failure Modes and Behavior

| Failure | Behavior | Recovery |
|---------|----------|----------|
| **Target service down** | New connections fail at dial; logged as ERROR; existing connections unaffected | Manual: Fix target service. GateRelay keeps retrying each new connection. |
| **GateRelay crash** | All active connections drop; service stops | Automatic: systemd restarts service (Restart=on-failure) |
| **Connection limit reached** | New connections rejected with WARN log; existing connections unaffected | Manual: Increase max_conns or wait for connections to close |
| **Config file invalid** | Service fails to start; logs error | Manual: Fix config and restart |
| **Port already in use** | Service fails to start; logs "address already in use" | Manual: Identify conflicting process, stop it, or change port |
| **Network partition** | Active connections stall or timeout; new connections may fail | Automatic: idle_timeout closes stale connections |

## Observability

### Metrics Available (via logs)

- **Total connections:** Count of "connection accepted" logs
- **Active connections:** Parse current state (accepted - closed)
- **Bytes transferred:** Sum of bytes_rx and bytes_tx fields
- **Error rate:** Count of ERROR level logs
- **Connection rejections:** Count of "connection limit reached" warnings
- **Target failures:** Count of "failed to connect to target" errors

### Future Enhancements

Potential additions (not in v1):
- Prometheus metrics endpoint
- Health check HTTP endpoint
- Real-time statistics API
- Connection rate limiting per source IP
- Support for multiple target backends (load balancing)

## Threat Model

### In-Scope Threats

1. **Unauthorized access to relay service**
   - Mitigation: Firewall rules, SSH hardening

2. **Privilege escalation from compromised service**
   - Mitigation: systemd sandboxing, no capabilities, syscall filtering

3. **Resource exhaustion (connection flood)**
   - Mitigation: max_conns limit, firewall rate limiting

4. **Configuration tampering**
   - Mitigation: File permissions (only root can write)

5. **Man-in-the-middle attacks**
   - Note: GateRelay itself doesn't do TLS (application layer concern)
   - Consider: Use TLS-aware target or terminate TLS at load balancer

### Out-of-Scope (Accepted Risks)

- **DDoS attacks:** Requires upstream mitigation (ISP, cloud provider)
- **Zero-day exploits in OS/kernel:** Keep system updated
- **Physical access to server:** Assumes trusted datacenter/lab
- **Data exfiltration via relayed traffic:** GateRelay is a relay, not a firewall; traffic content is application responsibility

## Compliance Considerations

### Audit Trail

All significant events are logged:
- Service start/stop
- Configuration loaded
- Every connection (source IP, timestamps, bytes)
- All errors

Logs are timestamped and tamper-evident (journald with sealing if enabled).

### Change Management

- Configuration changes require root access
- Service restart creates audit trail in journald
- Recommended: Version control config file, track changes

### Access Control

- SSH access: Key-only, admin users only
- Service account: No login shell, no sudo
- Config files: Only root can modify

## Testing and Validation

### Manual Testing Checklist

- [ ] Service starts successfully
- [ ] Logs show structured JSON output
- [ ] Client can connect to relay port
- [ ] Data flows through relay to target
- [ ] Connection closes gracefully
- [ ] Service survives target service restart
- [ ] Service auto-restarts after kill -9
- [ ] Firewall blocks unauthorized ports
- [ ] SSH login requires key (password fails)
- [ ] Connection limit enforcement works
- [ ] Graceful shutdown waits for connections

### Test Commands

```bash
# Test connectivity
nc -zv gaterelay-host 4000

# Test relay (if target is echo service)
echo "test" | nc gaterelay-host 4000

# Simulate crash
sudo kill -9 $(systemctl show -p MainPID gaterelay | cut -d= -f2)
sleep 5
sudo systemctl status gaterelay  # Should show: active (running)

# Test connection limit
for i in {1..250}; do nc gaterelay-host 4000 & done
sudo journalctl -u gaterelay -n 50 | grep "connection limit"
```

## References

- [systemd hardening documentation](https://www.freedesktop.org/software/systemd/man/systemd.exec.html)
- [UFW firewall guide](https://help.ubuntu.com/community/UFW)
- [Go slog package](https://pkg.go.dev/log/slog)
- [SSH hardening guide](https://www.ssh.com/academy/ssh/sshd_config)
- [OWASP Secure Coding Practices](https://owasp.org/www-project-secure-coding-practices-quick-reference-guide/)

## Version History

| Version | Date | Changes |
|---------|------|---------|
| 1.0.0 | 2026-01 | Initial production release |

## Contact and Support

For questions about this architecture or the GateRelay service:
- **Project Repository:** https://github.com/nathantebbs/gaterelay
- **Documentation:** See `deploy/docs/` directory
- **Issues:** GitHub issues or internal ticketing system
