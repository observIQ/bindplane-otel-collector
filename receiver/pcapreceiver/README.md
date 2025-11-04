# PCAP Receiver

The PCAP Receiver captures network packets and emits them as OpenTelemetry logs. It uses system-native command-line tools (`tcpdump` on macOS/Linux, `dumpcap` on Windows) to capture packets directly from a network interface.

## Supported Pipelines

- Logs

## Prerequisites

⚠️ **This receiver requires elevated privileges to capture network packets.**

### macOS

**Tool**: `tcpdump` is pre-installed on macOS. No additional installation required.

To verify:
```bash
which tcpdump
tcpdump --version
```

**Privileges**: Run the collector with `sudo`:
```bash
sudo /path/to/collector --config config.yaml
```

### Linux

**Tool**: `tcpdump` is pre-installed on Linux. No additional installation required.

To verify:
```bash
which tcpdump
tcpdump --version
```

**Privileges**: You can either run as root or use Linux capabilities.

- **Option A: Run as root**
  ```bash
  sudo /path/to/collector --config config.yaml
  ```

- **Option B: Grant capabilities to tcpdump** (common on many distros):
  ```bash
  sudo setcap cap_net_raw,cap_net_admin=eip /usr/sbin/tcpdump
  getcap /usr/sbin/tcpdump  # verify
  ```

- **Option C: Grant capabilities to the collector binary** (alternative):
  ```bash
  sudo setcap cap_net_raw,cap_net_admin=eip /path/to/collector
  ```

### Windows

**Tool**: Requires Wireshark (which includes Npcap). Ensure `dumpcap.exe` is available (typically from Wireshark installation, or specify `executable_path`).

- Install Wireshark: `https://www.wireshark.org/download.html` (includes Npcap during installation)
- List interfaces:
  ```powershell
  dumpcap.exe -D
  ```
- Optional: set explicit path in config:
  ```yaml
  receivers:
    pcap:
      interface: 1
      executable_path: "C:\\Program Files\\Wireshark\\dumpcap.exe"
  ```

**Privileges**: Run as Administrator if Npcap was installed in Admin-only mode, or reinstall Npcap without Admin-only mode to allow non-admin capture.

### Security Considerations

- Only run the collector as root when necessary for packet capture
- Use BPF filters to limit captured traffic and reduce security exposure
- Consider using a dedicated system user with minimal privileges for other collector components

## How It Works

1. The receiver spawns a capture tool process (`tcpdump` on macOS/Linux, `dumpcap` on Windows) with the specified interface and filter.
2. The capture tool outputs packet data in hex format (`-xx` flag for tcpdump, `-x` flag for dumpcap).
3. The receiver parses the text output to extract:
   - Network protocol (IP, IPv6, ARP)
   - Transport protocol (TCP, UDP, ICMP)
   - Source and destination addresses
   - Source and destination ports (when applicable)
   - Full packet data as hex string
4. Each packet is emitted as an OTel log with the hex-encoded packet as the body.

## Configuration

| Field         | Type   | Default | Required | Description                                                  |
| ------------- | ------ | ------- | -------- | ------------------------------------------------------------ |
| `interface`   | string | `any`* | No      | Network interface to capture packets from (e.g., `en0`, `eth0`). |
| `filter`      | string | `""`    | No       | BPF (Berkeley Packet Filter) expression to filter packets.    |
| `snaplen`     | int    | `65535` | No       | Maximum bytes to capture per packet (64-65535).             |
| `promiscuous` | bool   | `true`  | No       | Enable promiscuous mode to capture all network traffic.     |
| `executable_path` | string | `dumpcap` | No | Windows only, optional path to `dumpcap` executable. |
| `parse_attributes` | bool | `true` | No | Parse network attributes and add them to the logs. |

\* Defaults to `"1"` on Windows 

### Interface Names

To list available interfaces on macOS/Linux:
```bash
tcpdump -D
```

To list available interfaces on Windows:
```bash
dumpcap -D
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
    "network.type": "IP",
    "network.transport": "TCP",
    "source.address": "192.168.1.100",
    "destination.address": "192.168.1.1",
    "source.port": 54321,
    "destination.port": 443,
    "packet.length": 60
  }
}
```
**Note:** Attributes will only be parsed when `parse_attributes` is `true`.

### Attributes

- `network.type`: Network layer protocol (`IP`, `IPv6`, `ARP`)
- `network.transport`: Transport layer protocol (`TCP`, `UDP`, `ICMP`, or `Unknown`)
- `source.address`: Source IP address
- `destination.address`: Destination IP address
- `source.port`: Source port (omitted for ICMP and other non-port protocols)
- `destination.port`: Destination port (omitted for ICMP and other non-port protocols)
- `packet.length`: Total packet size in bytes

### Body Format

The log body contains the full packet data as a hex-encoded string with `0x` prefix. This can be decoded and analyzed downstream.

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

**Error**: `failed to start capture command: tcpdump: command not found` or `dumpcap.exe not found`

**Solution**:
- macOS: `tcpdump` should be pre-installed. Check `/usr/sbin/tcpdump`.
- Linux: Install: `apt install tcpdump` or `yum install tcpdump`.
- Windows: Install Wireshark (which includes Npcap) from `https://www.wireshark.org/download.html`. `dumpcap.exe` is typically located at `C:\Program Files\Wireshark\dumpcap.exe`. Ensure it's on PATH or configure `executable_path`.

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

