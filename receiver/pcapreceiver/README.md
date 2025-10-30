# PCAP Receiver

The PCAP Receiver captures network packets and emits them as OpenTelemetry logs. It uses system-native command-line tools (`tcpdump` on macOS/Linux) to capture packets.

## Supported Pipelines

- Logs

## Supported Platforms

- **macOS (darwin)**: ✅ Fully supported
- **Linux**: ⏳ Planned for future release  
- **Windows**: ⏳ Planned for future release

## How It Works

1. The receiver spawns a `tcpdump` process with the specified interface and filter
2. `tcpdump` outputs packet data in hex format (`-xx` flag)
3. The receiver parses the text output to extract:
   - Network protocol (IP, IPv6, ARP)
   - Transport protocol (TCP, UDP, ICMP)
   - Source and destination addresses
   - Source and destination ports (when applicable)
   - Full packet data as hex string
4. Each packet is emitted as an OTel log with the hex-encoded packet as the body and structured attributes

## Privilege Requirements

⚠️ **This receiver requires elevated privileges to capture network packets.**

### macOS

Run the collector with `sudo`:

```bash
sudo /path/to/collector --config config.yaml
```

### Why Root Privileges?

Packet capture requires access to network interfaces at a low level, which is restricted to privileged users for security reasons. On Unix-like systems, this requires running as root (UID 0).

### Security Considerations

- Only run the collector as root when necessary for packet capture
- Use BPF filters to limit captured traffic and reduce security exposure
- Consider using a dedicated system user with minimal privileges for other collector components
- In future releases, Linux support may include capability-based privileges (`CAP_NET_RAW`) as an alternative to full root access

## Configuration

| Field         | Type   | Default | Required | Description                                                  |
| ------------- | ------ | ------- | -------- | ------------------------------------------------------------ |
| `interface`   | string | `"en0"` | Yes      | Network interface to capture packets from (e.g., `en0`, `eth0`) |
| `filter`      | string | `""`    | No       | BPF (Berkeley Packet Filter) expression to filter packets    |
| `snaplen`     | int    | `65535` | No       | Maximum bytes to capture per packet (64-65535)               |
| `promiscuous` | bool   | `true`  | No       | Enable promiscuous mode to capture all network traffic       |

### Interface Names

Common interface names by platform:
- **macOS**: `en0` (Wi-Fi), `en1` (Ethernet), `lo0` (loopback)
- **Linux**: `eth0`, `wlan0`, `lo`

To list available interfaces on macOS:
```bash
tcpdump -D
```

### BPF Filters

BPF filters allow you to capture only specific traffic. Examples:

```yaml
# Capture only HTTPS traffic
filter: "tcp port 443"

# Capture DNS queries and responses
filter: "udp port 53"

# Capture HTTP and HTTPS
filter: "tcp port 80 or tcp port 443"

# Capture traffic to/from specific IP
filter: "host 192.168.1.100"

# Complex filter with multiple conditions
filter: "(tcp port 80 or tcp port 443) and not src 192.168.1.1"
```

BPF filter syntax reference: [tcpdump manual](https://www.tcpdump.org/manpages/pcap-filter.7.html)

## Example Configurations

### Basic Configuration

```yaml
receivers:
  pcap:
    interface: en0
```

### Capture DNS Traffic

```yaml
receivers:
  pcap:
    interface: en0
    filter: "udp port 53"
    snaplen: 1024
```

### Capture All HTTP/HTTPS Traffic

```yaml
receivers:
  pcap:
    interface: en0
    filter: "tcp port 80 or tcp port 443"
    promiscuous: true
```

## Output Format

Each captured packet is emitted as an OTel log with the following structure:

```json
{
  "timestamp": "2025-10-30T12:34:56.789012Z",
  "body": "0x4500003c1c4640004006b1e6c0a80164c0a80101d43101bb499602d200000000a002fffffe300000020405b40103030601010",
  "attributes": {
    "network.protocol": "IP",
    "network.transport": "TCP",
    "network.src.address": "192.168.1.100",
    "network.dst.address": "192.168.1.1",
    "network.src.port": 54321,
    "network.dst.port": 443,
    "packet.length": 60
  }
}
```

### Attributes

- `network.protocol`: Network layer protocol (`IP`, `IPv6`, `ARP`)
- `network.transport`: Transport layer protocol (`TCP`, `UDP`, `ICMP`, or `Unknown`)
- `network.src.address`: Source IP address
- `network.dst.address`: Destination IP address
- `network.src.port`: Source port (omitted for ICMP and other non-port protocols)
- `network.dst.port`: Destination port (omitted for ICMP and other non-port protocols)
- `packet.length`: Total packet size in bytes

### Body Format

The log body contains the full packet data as a hex-encoded string with `0x` prefix. This can be decoded and analyzed by downstream systems or Chronicle Forwarder.

## Prerequisites

### macOS

`tcpdump` is pre-installed on macOS. No additional installation required.

To verify:
```bash
which tcpdump
tcpdump --version
```

## Troubleshooting

### "permission denied" Error

**Error**: `failed to start capture command: permission denied`

**Solution**: Run the collector with `sudo`:
```bash
sudo /path/to/collector --config config.yaml
```

### "tcpdump: command not found"

**Error**: `failed to start capture command: tcpdump: command not found`

**Solution**: Install tcpdump:
- macOS: `tcpdump` should be pre-installed. Check `/usr/sbin/tcpdump`
- If missing, reinstall Command Line Tools: `xcode-select --install`

### "No such device exists"

**Error**: `tcpdump: en0: No such device exists`

**Solution**: The specified interface doesn't exist. List available interfaces:
```bash
tcpdump -D
# or
ifconfig
```

Update the `interface` field in your configuration with a valid interface name.

### No Packets Captured

If the receiver starts but no packets appear:

1. **Check BPF filter**: Ensure your filter matches actual traffic
   ```bash
   # Test filter manually
   sudo tcpdump -i en0 -c 10 "tcp port 443"
   ```

2. **Verify interface is active**: Ensure the interface has traffic
   ```bash
   # Generate test traffic
   ping google.com
   ```

3. **Check promiscuous mode**: Some interfaces may not support promiscuous mode. Try setting `promiscuous: false`

### High CPU Usage

If packet capture causes high CPU usage:

1. **Use specific BPF filters**: Capture only needed traffic
2. **Reduce snaplen**: Capture only packet headers: `snaplen: 128`
3. **Add processors**: Use sampling or filtering processors downstream

## Performance Considerations

- **CPU**: Packet parsing is CPU-intensive. Use BPF filters to limit captured traffic.
- **Memory**: Each packet creates an OTel log record. Use batch processors to manage memory.
- **Disk I/O**: High packet rates can generate significant log volume. Consider sampling.
- **Network**: Capturing on busy interfaces (e.g., production servers) may impact performance.

## Limitations

- Currently only macOS (darwin) is supported
- Requires root privileges (cannot run as unprivileged user)
- No built-in rate limiting (use downstream processors)
- Does not reassemble fragmented packets
- Does not decode application-layer protocols (HTTP, DNS, etc.) - only provides raw packet data

## Future Enhancements

- Linux support with `tcpdump`
- Windows support with `windump` / `npcap`
- Capability-based privileges on Linux (`CAP_NET_RAW`)
- Optional packet reassembly
- Built-in rate limiting / sampling

