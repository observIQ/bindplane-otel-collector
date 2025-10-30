# PCAP Receiver

The PCAP Receiver captures network packets and emits them as OpenTelemetry logs. It uses system-native command-line tools (`tcpdump` on macOS/Linux) to capture packets, making it a pure-Go solution that doesn't require cgo or native library dependencies.

## Minimum Agent Versions

- Introduced: v1.x.x (TBD)

## Supported Pipelines

- Logs

## Supported Platforms

- **macOS (darwin)**: ✅ Fully supported
- **Linux**: ✅ Fully supported
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

### Linux

You can either run as root or use Linux capabilities.

- Option A: Run as root

```bash
sudo /path/to/collector --config config.yaml
```

- Option B: Grant capabilities to tcpdump (common on many distros):

```bash
sudo setcap cap_net_raw,cap_net_admin=eip /usr/sbin/tcpdump
getcap /usr/sbin/tcpdump  # verify
```

- Option C: Grant capabilities to the collector binary (alternative):

```bash
sudo setcap cap_net_raw,cap_net_admin=eip /path/to/collector
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
- **Linux**: `eth0`, `ens160`, `wlan0`, `lo`

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
    filter: "tcp port 443"

exporters:
  debug:
    verbosity: detailed

service:
  pipelines:
    logs:
      receivers: [pcap]
      exporters: [debug]
```

### Capture DNS Traffic

```yaml
receivers:
  pcap:
    interface: en0
    filter: "udp port 53"
    snaplen: 1024

exporters:
  chronicle_forwarder:
    endpoint: "forwarder.example.com:514"

service:
  pipelines:
    logs:
      receivers: [pcap]
      exporters: [chronicle_forwarder]
```

### Capture All HTTP/HTTPS Traffic

```yaml
receivers:
  pcap:
    interface: en0
    filter: "tcp port 80 or tcp port 443"
    promiscuous: true

processors:
  batch:
    timeout: 10s
    send_batch_size: 100

exporters:
  otlp:
    endpoint: collector.example.com:4317

service:
  pipelines:
    logs:
      receivers: [pcap]
      processors: [batch]
      exporters: [otlp]
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

### Linux

`tcpdump` is pre-installed on Linux. No additional installation required.

To verify:
```bash
which tcpdump
tcpdump --version
```

### Windows

Requires Npcap (WinPcap-compatible). Ensure `windump.exe` is available (on PATH or specify `executable_path`).

- Install Npcap: `https://nmap.org/npcap/`
- List interfaces:

```powershell
windump.exe -D
```

- Run as Administrator if Npcap was installed in Admin-only mode, or reinstall Npcap without Admin-only mode to allow non-admin capture.

- Optional: set explicit path in config:

```yaml
receivers:
  pcap:
    interface: 1
    executable_path: "C:\\Program Files\\Npcap\\windump.exe"
```

### Running the Collector

```bash
# Download and install the collector
# ... installation steps ...

# Run with sudo for packet capture privileges
sudo /path/to/collector --config /path/to/config.yaml
```

## Troubleshooting

### "permission denied" Error

**Error**: `failed to start capture command: permission denied`

**Solution**:
- macOS: run the collector with `sudo`.
- Linux: run as root or grant capabilities:

```bash
sudo setcap cap_net_raw,cap_net_admin=eip /usr/sbin/tcpdump
getcap /usr/sbin/tcpdump
```

### "tcpdump: command not found"

**Error**: `failed to start capture command: tcpdump: command not found` or `windump.exe not found`

**Solution**:
- macOS: `tcpdump` should be pre-installed. Check `/usr/sbin/tcpdump`.
- Linux: Install: `apt install tcpdump` or `yum install tcpdump`.
- Windows: Install Npcap and ensure `windump.exe` is on PATH or configure `executable_path`.

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

- macOS and Linux supported; Windows not yet supported
- Requires elevated privileges (root or capabilities)
- No built-in rate limiting (use downstream processors)
- Does not reassemble fragmented packets
- Does not decode application-layer protocols (HTTP, DNS, etc.) - only provides raw packet data

## Future Enhancements

- Windows support with `windump` / `npcap`
- Capability-based guidance improvements and auto-detection
- Optional packet reassembly
- Built-in rate limiting / sampling
