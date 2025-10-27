# PCAP Receiver

The PCAP Receiver captures network packets from interfaces and converts packet metadata to OpenTelemetry logs.

## Requirements

- **Linux**: Root privileges or `CAP_NET_RAW` capability
- **macOS**: Root privileges or BPF device access entitlement
- **Windows**: Administrator privileges and [Npcap](https://npcap.com/) installed

## Configuration

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `interface` | string | "" | Network interface to capture from. If empty, auto-detects first available interface |
| `snap_length` | int32 | 65535 | Maximum bytes to capture per packet |
| `bpf_filter` | string | "" | Berkeley Packet Filter expression to filter packets (e.g., "tcp port 80") |
| `promiscuous` | bool | true | Enable promiscuous mode on the interface |

## Example Configuration

### Basic Configuration

```yaml
receivers:
  pcap:
    interface: eth0
    snap_length: 65535
    promiscuous: true
```

### With BPF Filter

```yaml
receivers:
  pcap:
    interface: eth0
    bpf_filter: "tcp port 80 or tcp port 443"
```

### Complete Pipeline

```yaml
receivers:
  pcap:
    interface: eth0
    bpf_filter: "tcp"
    
exporters:
  file:
    path: ./packets.json
    format: json

service:
  pipelines:
    logs:
      receivers: [pcap]
      exporters: [file]
```

## Platform-Specific Notes

### Linux

Requires root privileges or `CAP_NET_RAW` capability:

```bash
# Grant capability
sudo setcap cap_net_raw+ep /path/to/collector

# Or run as root
sudo ./collector --config config.yaml
```

### macOS

Requires root privileges:

```bash
sudo ./collector --config config.yaml
```

### Windows

Requires Administrator privileges and Npcap installation:

1. Install [Npcap](https://npcap.com/#download)
2. Run PowerShell as Administrator
3. Execute the collector


## BPF Filter Examples

Berkeley Packet Filter (BPF) syntax allows fine-grained packet filtering:

```yaml
# Capture only HTTP traffic
bpf_filter: "tcp port 80"

# Capture HTTP and HTTPS traffic
bpf_filter: "tcp port 80 or tcp port 443"

# Capture DNS queries
bpf_filter: "udp port 53"

# Capture traffic to/from specific host
bpf_filter: "host 192.168.1.100"

# Capture traffic on specific subnet
bpf_filter: "net 10.0.0.0/8"

# Complex filter combining multiple conditions
bpf_filter: "tcp and (port 80 or port 443) and host 192.168.1.100"
```

