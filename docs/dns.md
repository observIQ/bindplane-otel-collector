# DNS Resolver Configuration

The Bindplane Agent supports custom DNS resolver configuration to control how DNS queries are resolved throughout the application. This feature should only be used if the host operating system's DNS resolver is misbehaving or should be otherwise ignored. It allows you to specify a custom DNS server and timeout settings for DNS lookups.

## Overview

The DNS resolver configuration affects all DNS lookups performed by the Bindplane Collector, including:
- OpAMP WebSocket connections to management platforms
- HTTP/gRPC connections to data export endpoints (Chronicle, Google Cloud, etc.)
- Any other DNS queries made by the application

## Configuration Methods

The DNS resolver can be configured using either a configuration file or environment variables. **You cannot combine both methods** - only one configuration method can be used at a time.

### Method 1: Configuration File

Use the `--resolver` command-line flag or the `RESOLVER_YAML_PATH` environment variable to specify a YAML configuration file.

#### Command-line Flag
```bash
./bindplane-otel-collector --resolver /path/to/resolver-config.yaml
```

#### Environment Variable
```bash
export RESOLVER_YAML_PATH="/path/to/resolver-config.yaml"
./bindplane-otel-collector
```

#### Configuration File Format
```yaml
server: "8.8.8.8:53"
timeout: 5s
```

**Configuration Parameters:**
- `server` (required): DNS server address in "host:port" format
- `timeout` (required): DNS query timeout duration (must be greater than 0)

**Example Configuration Files:**

Minimal configuration:
```yaml
server: "1.1.1.1:53"
timeout: 2s
```

Custom port configuration:
```yaml
server: "9.9.9.9:5353"
timeout: 10s
```

IPv6 DNS server:
```yaml
server: "[2001:4860:4860::8888]:53"
timeout: 5s
```

### Method 2: Environment Variables

Configure the DNS resolver using environment variables. This method requires all three environment variables to be set.

#### Required Environment Variables
```bash
export BINDPLANE_OTEL_COLLECTOR_RESOLVER_ENABLE="true"
export BINDPLANE_OTEL_COLLECTOR_RESOLVER_SERVER="8.8.8.8:53"
export BINDPLANE_OTEL_COLLECTOR_RESOLVER_TIMEOUT="5s"
```

**Environment Variable Details:**
- `BINDPLANE_OTEL_COLLECTOR_RESOLVER_ENABLE`: Must be set to "true" to enable resolver configuration
- `BINDPLANE_OTEL_COLLECTOR_RESOLVER_SERVER`: DNS server address in "host:port" format
- `BINDPLANE_OTEL_COLLECTOR_RESOLVER_TIMEOUT`: DNS query timeout duration (e.g., "5s", "2m", "500ms")

#### Example Environment Variable Configurations

Basic configuration:
```bash
export BINDPLANE_OTEL_COLLECTOR_RESOLVER_ENABLE="true"
export BINDPLANE_OTEL_COLLECTOR_RESOLVER_SERVER="1.1.1.1:53"
export BINDPLANE_OTEL_COLLECTOR_RESOLVER_TIMEOUT="3s"
```

Custom port configuration:
```bash
export BINDPLANE_OTEL_COLLECTOR_RESOLVER_ENABLE="true"
export BINDPLANE_OTEL_COLLECTOR_RESOLVER_SERVER="9.9.9.9:5353"
export BINDPLANE_OTEL_COLLECTOR_RESOLVER_TIMEOUT="10s"
```

IPv6 configuration:
```bash
export BINDPLANE_OTEL_COLLECTOR_RESOLVER_ENABLE="true"
export BINDPLANE_OTEL_COLLECTOR_RESOLVER_SERVER="[2001:4860:4860::8888]:53"
export BINDPLANE_OTEL_COLLECTOR_RESOLVER_TIMEOUT="5s"
```

## Important Notes

### Configuration Method Exclusivity
**You cannot combine file and environment variable configuration methods.** The resolver will use the configuration file if the `--resolver` flag is provided or if `RESOLVER_YAML_PATH` is set. If neither is provided, it will fall back to environment variables.

### Timeout Requirements
- Timeout values must be positive (greater than 0)
- Supported formats: "5s", "500ms", etc.
- Invalid timeout values will cause the collector to fail to start

### DNS Server Format
- Use "host:port" format (e.g., "8.8.8.8:53")
- For IPv6 addresses, use brackets: "[2001:4860:4860::8888]:53"
- Default DNS port is 53 if not specified

### Logging
When DNS resolver configuration is enabled, the collector will log:
- DNS resolver configuration success/failure during startup
- Debug-level logs for each DNS connection attempt
- DNS connection success/failure for troubleshooting

#### Example DNS Resolver Logs
```json
{
  "level": "debug",
  "ts": "2025-10-15T15:06:14.851-0400",
  "logger": "resolver",
  "caller": "resolver/resolver.go:146",
  "msg": "DNS resolver dial successful",
  "server": "1.1.1.1:53",
  "timeout_seconds": 2,
  "network": "udp",
  "original_server": "10.33.40.3:53"
}
```

**Log Field Explanations:**
- `server`: The configured DNS server being used (e.g., "1.1.1.1:53")
- `original_server`: The DNS server that would have been used if the resolver was not configured (e.g., "10.33.40.3:53")
- `network`: The network protocol used for the DNS query ("udp" or "tcp")
- `timeout_seconds`: The configured timeout duration in seconds
