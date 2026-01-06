# GateRelay

**Secure Linux TCP Relay Service** - A production-grade TCP relay demonstrating enterprise systems administration practices.

[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

## Overview

GateRelay is a hardened TCP relay service designed to forward connections from a public ingress port to a controlled egress target. Built as a portfolio project to demonstrate **Sr. Systems Administrator** competencies in:

- Building repeatable, production-ready systems
- Running services with least privilege
- Implementing defense-in-depth security
- Operating with comprehensive observability
- Recovering from failures gracefully

**Use Case:** Forward traffic from a public-facing port to an internal/backend service, with full operational rigor and security hardening.

## Key Features

### Application
- Bidirectional TCP relay with configurable listen and target endpoints
- Connection limits and timeout management
- Graceful shutdown with configurable grace period
- Structured JSON logging (via Go's `log/slog`)
- Comprehensive connection statistics

### Security
- Runs as dedicated non-root user via systemd
- Extensive systemd sandboxing (NoNewPrivileges, ProtectSystem, capability dropping, syscall filtering)
- Firewall configuration (ufw) with minimal exposure
- SSH hardening scripts (key-only auth, no root login)
- Read-only binary and config, isolated state directory

### Operations
- Automated installation and deployment scripts
- systemd service with auto-restart on failure
- Complete observability via journald
- Detailed runbook with failure scenarios and recovery procedures
- Architecture documentation with threat model

## Quick Start

### Prerequisites

- Ubuntu Server 22.04 LTS or Debian 12 Stable
- Go 1.21+ (for building from source)
- SSH access with sudo privileges

### 1. Build

```bash
git clone <repo-url>
cd gate-relay
go build -o gaterelay
```

### 2. Deploy

Transfer to your VM and run installation:

```bash
# On deployment VM
sudo ./deploy/scripts/install.sh
```

### 3. Configure

Edit configuration:

```bash
sudo nano /etc/gaterelay/config.toml
```

Example configuration:
```toml
listen_addr = "0.0.0.0"
listen_port = 4000
target_addr = "10.0.0.20"
target_port = 5000
max_conns = 200
idle_timeout_secs = 60
connect_timeout_secs = 5
shutdown_grace_secs = 10
log_level = "info"
```

### 4. Harden

```bash
# Configure firewall
sudo ./deploy/scripts/setup-firewall.sh 4000 22 192.168.1.0/24

# Harden SSH (ensure you have key auth set up first!)
sudo ./deploy/scripts/harden-ssh.sh
```

### 5. Start

```bash
sudo systemctl start gaterelay
sudo systemctl enable gaterelay
sudo systemctl status gaterelay
```

### 6. Monitor

```bash
# View logs
sudo journalctl -u gaterelay -f

# Check connections
sudo ss -tn | grep :4000
```

## Project Structure

```
gate-relay/
├── main.go                 # Service entry point
├── config.go               # Configuration parsing and validation
├── relay.go                # Core TCP relay implementation
├── go.mod                  # Go module definition
├── go.sum                  # Go module checksums
├── gaterelay               # Compiled binary (after build)
├── README.md               # This file
├── prompts/                # Project specification
│   └── project_1_*.md
└── deploy/                 # Deployment artifacts
    ├── config/
    │   └── config.toml     # Example configuration
    ├── systemd/
    │   └── gaterelay.service   # systemd unit file
    ├── scripts/
    │   ├── install.sh          # Installation automation
    │   ├── setup-firewall.sh   # Firewall configuration
    │   └── harden-ssh.sh       # SSH hardening
    └── docs/
        ├── DEPLOYMENT.md       # Step-by-step deployment guide
        ├── OPERATIONS.md       # Operations runbook
        └── ARCHITECTURE.md     # Technical architecture
```

## Documentation

| Document | Description |
|----------|-------------|
| [DEPLOYMENT.md](deploy/docs/DEPLOYMENT.md) | Complete deployment guide from scratch |
| [OPERATIONS.md](deploy/docs/OPERATIONS.md) | Day-to-day operations, monitoring, and failure recovery |
| [ARCHITECTURE.md](deploy/docs/ARCHITECTURE.md) | Technical architecture, security design, and threat model |
| [HARDENING.md](deploy/docs/HARDENING.md) | Security hardening checklist |

## Success Criteria

This project demonstrates the following competencies:

✅ **Repeatable Deployment**: A third party can rebuild the system using provided documentation

✅ **Least Privilege**: Service runs as non-root with minimal capabilities

✅ **Defense in Depth**: Multiple security layers (firewall, systemd, application)

✅ **Observability**: Structured logs for all significant events

✅ **Resilience**: Automatic recovery from common failures

✅ **Documentation**: Complete architecture, deployment, and operations guides


### Demonstrated Failure Scenarios

1. **Target service down** → GateRelay logs errors, continues accepting connections
2. **Service crash (kill -9)** → systemd automatically restarts service
3. **Port scan / unauthorized access** → Firewall blocks and logs attempts

## Technology Stack

- **Language:** Go 1.21+
- **Configuration:** TOML (via `github.com/BurntSushi/toml`)
- **Logging:** Structured JSON (Go `log/slog`)
- **Service Management:** systemd
- **Firewall:** ufw (Uncomplicated Firewall)
- **Platform:** Ubuntu Server 22.04 LTS / Debian 12

## Security Highlights

### systemd Hardening
- `NoNewPrivileges=true`
- `ProtectSystem=strict`
- `PrivateTmp=true`
- `CapabilityBoundingSet=` (empty)
- `RestrictAddressFamilies=AF_INET AF_INET6`
- `SystemCallFilter=@system-service`
- And [many more](deploy/systemd/gaterelay.service)...

### Network Security
- Default deny firewall policy
- Only required ports exposed
- SSH restricted to admin network (configurable)
- Rate limiting on SSH

### Access Control
- Dedicated service user (`gaterelay`) with no shell
- Config files owned by root, read-only to service
- State directory owned by service user, isolated

## Usage Examples

### Basic Operation

```bash
# Start service
sudo systemctl start gaterelay

# Check status
sudo systemctl status gaterelay

# View logs
sudo journalctl -u gaterelay -f

# Stop service
sudo systemctl stop gaterelay
```

### Testing Connectivity

```bash
# From client machine
nc -zv gaterelay-host 4000

# Send data through relay
echo "test message" | nc gaterelay-host 4000
```

### Monitoring

```bash
# Count active connections
sudo ss -tn | grep :4000 | wc -l

# Count total connections today
sudo journalctl -u gaterelay --since today | grep -c "connection accepted"

# Check for errors
sudo journalctl -u gaterelay --since today | grep '"level":"ERROR"'
```

## Operations Runbook

### Common Tasks

**Update configuration:**
```bash
sudo nano /etc/gaterelay/config.toml
sudo systemctl restart gaterelay
```

**Update binary:**
```bash
sudo systemctl stop gaterelay
sudo install -m 755 gaterelay /usr/local/bin/gaterelay
sudo systemctl start gaterelay
```

**View recent activity:**
```bash
sudo journalctl -u gaterelay --since "1 hour ago" -o json-pretty
```

### Troubleshooting

**Service won't start:**
```bash
sudo journalctl -u gaterelay -n 100
sudo /usr/local/bin/gaterelay -config /etc/gaterelay/config.toml
```

**Target unreachable:**
```bash
nc -zv target-host target-port
sudo journalctl -u gaterelay | grep "failed to connect"
```

For detailed procedures, see [OPERATIONS.md](deploy/docs/OPERATIONS.md).

## Performance

**Typical resource usage** (1 vCPU, 512MB RAM VM):

| Connections | Memory | CPU |
|-------------|--------|-----|
| Idle | ~5-8 MB | <1% |
| 50 active | ~12 MB | 2-5% |
| 200 active | ~20 MB | 5-15% |

**Throughput:** Saturates 1Gbps network with large transfers.

## Roadmap / Future Enhancements

Potential improvements (not in v1):
- [ ] Prometheus metrics endpoint
- [ ] HTTP health check endpoint
- [ ] TLS support for encrypted relay
- [ ] Load balancing across multiple targets
- [ ] Per-source-IP rate limiting
- [ ] Automated integration tests
- [ ] Docker/container deployment option

## Contributing

This is a portfolio project, but suggestions and improvements are welcome!

1. Fork the repository
2. Create a feature branch
3. Make your changes with tests
4. Submit a pull request

## License

MIT License - see [LICENSE](LICENSE) file for details.

## Author

**Nathan Tebbs**

Portfolio project demonstrating systems administration and operational excellence for senior-level roles.

## Acknowledgments

- Inspired by real-world production relay patterns
- systemd hardening based on [systemd security documentation](https://www.freedesktop.org/software/systemd/man/systemd.exec.html)
- Security practices aligned with [OWASP guidelines](https://owasp.org/)

## Additional Resources

- [Go security best practices](https://go.dev/doc/security/best-practices)
- [Linux hardening checklist](https://github.com/imthenachoman/How-To-Secure-A-Linux-Server)

**Questions? See the [documentation](deploy/docs/) or open an issue.**
