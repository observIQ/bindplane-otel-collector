# PCAP Receiver

The PCAP Receiver captures network packets from interfaces and converts packet metadata to OpenTelemetry logs. This receiver achieves functional parity with Google Chronicle Forwarder.

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

## Packet Attributes

Each captured packet is converted to a log record with the following attributes:

- `interface`: Source interface name
- `packet.size`: Total packet length in bytes
- `packet.timestamp`: Capture timestamp (RFC3339)
- `link.type`: Link layer type (e.g., "Ethernet")
- `link.src_addr`: Source MAC address
- `link.dst_addr`: Destination MAC address
- `network.protocol`: Network protocol (e.g., "IPv4", "IPv6")
- `network.src_addr`: Source IP address
- `network.dst_addr`: Destination IP address
- `transport.protocol`: Transport protocol (e.g., "TCP", "UDP", "ICMP")
- `transport.src_port`: Source port (TCP/UDP only)
- `transport.dst_port`: Destination port (TCP/UDP only)
- `application.protocol`: Application protocol (port-based detection)

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

## Chronicle Forwarder Parity

This receiver implements the same packet capture semantics as Google Chronicle Forwarder:

- Cross-platform support (Windows/Npcap, Linux/libpcap, macOS/BPF)
- Identical attribute schema for packet metadata
- BPF filter support
- Promiscuous mode support
- Full packet capture with configurable snap length

## Limitations

- Requires elevated privileges (root/admin)
- Live capture only (no PCAP file replay in initial version)
- Performance may be impacted by high packet rates

