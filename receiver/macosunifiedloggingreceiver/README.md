# macOS Unified Logging Receiver

The macOS Unified Logging Receiver collects logs from macOS systems using the native `log` command. This receiver supports both live system logs and archived log files (`.logarchive`).

## Key Features

- **Native Integration**: Uses the macOS `log show` command with NDJSON output
- **Archive Support**: Read from archived log files (`.logarchive` directories)
- **Live Mode**: Stream logs from the live system in real-time
- **Filtering**: Apply predicates to filter logs by subsystem, category, process, etc.
- **Time Range**: Specify start and end times for log collection

## Configuration

### Basic Configuration (Live Mode)

```yaml
receivers:
  macosunifiedlogging:
    poll_interval: 30s      # How often to poll for new logs
    max_log_age: 24h        # How far back to read on startup
```

### Archive Mode

```yaml
receivers:
  macosunifiedlogging:
    archive_path: "/path/to/system_logs.logarchive"
    start_time: "2024-01-01 00:00:00"
    end_time: "2024-01-02 00:00:00"
```

### With Filtering

```yaml
receivers:
  macosunifiedlogging:
    archive_path: "./logs.logarchive"
    predicate: "subsystem == 'com.apple.systempreferences'"
```

## Configuration Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `archive_path` | string | "" | Path to `.logarchive` directory. If empty, reads live system logs |
| `predicate` | string | "" | Filter predicate (e.g., `"subsystem == 'com.apple.example'"`) |
| `start_time` | string | "" | Start time in format "2006-01-02 15:04:05" |
| `end_time` | string | "" | End time (archive mode only) in format "2006-01-02 15:04:05" |
| `poll_interval` | duration | 30s | How often to poll for new logs (live mode only) |
| `max_log_age` | duration | 24h | Maximum age of logs to read on startup (live mode only) |

## Predicate Examples

Filter by subsystem:
```
subsystem == 'com.apple.systempreferences'
```

Filter by process:
```
process == 'kernel'
```

Filter by message type:
```
messageType == 'Error'
```

Combine filters:
```
subsystem == 'com.apple.example' AND messageType IN {'Error', 'Fault'}
```

## Output Format

The receiver converts macOS unified logging format to OpenTelemetry log records:

- **Body**: Contains the `eventMessage` field
- **Timestamp**: Parsed from the `timestamp` field
- **Severity**: Mapped from `messageType` (Error, Fault, Default, Info, Debug)
- **Attributes**: All other fields from the log command output

## Requirements

- macOS 10.12 (Sierra) or later
- The `log` command must be available in PATH
- For archive mode: Read access to the `.logarchive` directory
- For live mode: Appropriate permissions to read system logs

## Example

Complete example configuration:

```yaml
receivers:
  macosunifiedlogging:
    archive_path: "./system_logs.logarchive"
    predicate: "subsystem BEGINSWITH 'com.apple'"
    start_time: "2024-01-01 00:00:00"

exporters:
  debug:
    verbosity: detailed
  file:
    path: "./output/logs.json"
    format: json

service:
  pipelines:
    logs:
      receivers: [macosunifiedlogging]
      exporters: [debug, file]
```

