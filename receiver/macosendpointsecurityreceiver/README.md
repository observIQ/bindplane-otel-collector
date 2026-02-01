# macOS Endpoint Security Receiver

The macOS Endpoint Security Receiver collects Endpoint Security events from macOS systems using the native `eslogger` command. This receiver streams Endpoint Security events in real-time using JSON format.

## Requirements

- macOS 10.15 (Catalina) or later (Endpoint Security framework requirement)
- The `eslogger` command must be available in PATH
- **Root/sudo privileges** - eslogger requires super-user privileges
- **Full Disk Access TCC authorization** - The responsible process must have Full Disk Access authorization
  - For SSH sessions: Enable "Allow full disk access for remote users" in System Preferences > Sharing > Remote Login
  - For Terminal.app: Authorize Terminal.app in System Preferences > Security & Privacy > Privacy > Full Disk Access
  - For launch daemons: Authorize eslogger itself in System Preferences > Security & Privacy > Privacy > Full Disk Access

## Configuration

### Configuration Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `event_types` | []string | `["exec", "fork", "exit"]` | List of event types to subscribe to (e.g., `["exec", "fork", "exit", "open", "close"]`). See [Event Types](#event-types) below |
| `select_paths` | []string | `[]` | List of program path prefixes for filtering events. Events will only be captured for processes matching these paths (e.g., `["/bin/zsh", "/usr/bin"]`) |
| `max_poll_interval` | duration | 30s | Maximum interval for health checks/reconnection attempts. Note: eslogger streams continuously, so this is mainly for error recovery |

### Event Types

The receiver supports all Endpoint Security event types. Common event types include:

- **Process Lifecycle**: `exec`, `fork`, `exit`
- **File Operations**: `open`, `close`, `create`, `unlink`, `rename`, `readlink`, `link`
- **File System**: `mount`, `unmount`, `remount`, `chdir`, `chroot`
- **Network**: `uipc_bind`, `uipc_connect`
- **Security**: `authentication`, `authorization_judgement`, `authorization_petition`, `tcc_modify`
- **System**: `kextload`, `kextunload`, `signal`, `setuid`, `setgid`

To see all available event types, run:
```bash
eslogger --list-events
```

### Basic Configuration

```yaml
receivers:
  macosendpointsecurity:
    event_types: ["exec", "fork", "exit"]
    max_poll_interval: 30s
```

### With Path Filtering

```yaml
receivers:
  macosendpointsecurity:
    event_types: ["exec", "fork", "exit", "open", "close"]
    select_paths: ["/bin/zsh", "/usr/bin"]
```

### Comprehensive Security Monitoring

```yaml
receivers:
  macosendpointsecurity:
    event_types:
      - exec
      - fork
      - exit
      - open
      - close
      - create
      - unlink
      - rename
      - authentication
      - authorization_judgement
      - tcc_modify
      - kextload
      - kextunload
```

## Output Format

The receiver converts Endpoint Security events to OpenTelemetry log records:

- **Body**: Contains the entire event as a JSON string
- **Timestamp**: Parsed from the event timestamp field
- **Severity**: Set to Info level (security events are informational)
- **Attributes**: Not set (all event data is in the JSON body)

## Example

Complete example configuration:

```yaml
receivers:
  macosendpointsecurity:
    event_types: ["exec", "fork", "exit", "open", "close"]
    select_paths: ["/bin/zsh"]

exporters:
  file:
    path: "./output/security_events.json"
    format: json

service:
  pipelines:
    logs:
      receivers: [macosendpointsecurity]
      exporters: [file]
```

## Security Considerations

### Privileges Required

- **Root/sudo**: eslogger must be run with super-user privileges
- **Full Disk Access**: The responsible process must have Full Disk Access TCC authorization

### Process Group Suppression

eslogger automatically suppresses events for all processes that are part of its process group. This prevents feedback loops when filtering output using shell pipelines.

### Production Usage

⚠️ **Warning**: Running eslogger in production requires careful consideration:

- Ensure the collector process has appropriate privileges
- Monitor resource usage as Endpoint Security events can be high-volume
- Consider using `select_paths` to filter events and reduce volume
- Be aware that eslogger is not an API and may change between macOS releases

## Type Safety

This receiver uses type-safe event types with compile-time validation:

- Event types are defined as constants (e.g., `EventTypeExec`, `EventTypeFork`)
- Configuration validation ensures only valid event types are accepted
- Type-safe parsing interfaces provide structured access to event data

## Limitations

- **No Archive Mode**: eslogger only supports live streaming, not archived events
- **No Time Filtering**: Events are streamed in real-time only
- **JSON Format Only**: eslogger only supports JSON output format
- **Root Required**: Must run with root/sudo privileges
- **TCC Authorization**: Requires Full Disk Access authorization

## See Also

- [Endpoint Security Framework](https://developer.apple.com/documentation/endpointsecurity)
