# macOS Unified Logging Receiver

The macOS Unified Logging Receiver collects logs from macOS systems using the native `log` command. This receiver supports both live system logs and archived log files (`.logarchive`).

## Requirements

- macOS 10.12 (Sierra) or later
- The `log` command must be available in PATH
- For archive mode: Read access to the `.logarchive` directory
- For live mode: Appropriate permissions to read system logs

## Configuration

### Configuration Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `archive_path` | string | "" | Path to `.logarchive` directory. If empty, reads live system logs |
| `predicate` | string | "" | Filter predicate (e.g., `"subsystem == 'com.apple.example'"`) |
| `start_time` | string | "" | Start time in format "2006-01-02 15:04:05" |
| `end_time` | string | "" | End time in format "2006-01-02 15:04:05" (archive mode only) |
| `poll_interval` | duration | 30s | How often to poll for new logs (live mode only) |
| `max_log_age` | duration | 24h | Maximum age of logs to read on startup (live mode only) |
| `raw` | bool | false | Send log lines as stringBody instead of parsing as structured NDJSON |

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

### Raw Mode

```yaml
receivers:
  macosunifiedlogging:
    raw: true               
    poll_interval: 30s
    max_log_age: 24h
```


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

For a full description of predicate expressions, run `log help predicates`.

## Output Format

The receiver converts macOS logs to OpenTelemetry log records:

- **Body**: Contains the entire log line as a string (JSON format when `raw: false`, plain text when `raw: true`)
- **Attributes**: Not set

### Default Mode (JSON)

When `raw` is `false` (default), the log command outputs structured NDJSON. Each log line is captured as a complete JSON string in the body, with timestamp and severity extracted:

- **Timestamp**: Parsed from the `timestamp` field in the JSON
- **Severity**: Mapped from `messageType` (Error, Fault, Default, Info, Debug)

### Raw Mode

When `raw` is `true`, the log command outputs plain text format. Each log line is captured as plain text in the body:

- **Timestamp**: Set to observed time (when the log was received)
- **Severity**: Not set

## Example

Complete example configuration:

```yaml
receivers:
  macosunifiedlogging:
    archive_path: "./system_logs.logarchive"
    predicate: "subsystem BEGINSWITH 'com.apple'"
    start_time: "2024-01-01 00:00:00"

exporters:
  file:
    path: "./output/logs.json"
    format: json

service:
  pipelines:
    logs:
      receivers: [macosunifiedlogging]
      exporters: [file]
```

