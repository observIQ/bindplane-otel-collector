# macOS Unified Logging Receiver

This receiver reads macOS Unified Logging traceV3 files and converts them to OpenTelemetry log format using encoding extensions.

## Overview

The macOS Unified Logging system was introduced in macOS 10.12 (Sierra) as Apple's centralized logging system. It stores log data in binary files called traceV3 files, which contain compressed log entries with metadata.

This receiver:
- Watches for traceV3 files using the Stanza file consumer framework
- Uses encoding extensions to decode the binary traceV3 format
- Converts the decoded logs to OpenTelemetry log format
- Supports file watching and real-time processing

## Status

**⚠️ EXPERIMENTAL** - This receiver is currently under development. The encoding extension for traceV3 files is not yet implemented.

## Configuration

The receiver requires an encoding extension to decode traceV3 files.

### Basic Configuration

```yaml
extensions:
  macosunifiedlogencoding:
    # Configuration options TBD

receivers:
  macosunifiedloggingreceiver:
    encoding_extension: "macosunifiedlogencoding"
    include:
      - "/var/db/diagnostics/Persist/*.tracev3"
      - "/var/db/diagnostics/Special/*.tracev3"
    start_at: beginning
    
exporters:
  debug:
    verbosity: detailed

service:
  extensions: [macosunifiedlogencoding]
  pipelines:
    logs:
      receivers: [macosunifiedloggingreceiver]
      exporters: [debug]
```

### Configuration Parameters

| Parameter | Default | Description |
|-----------|---------|-------------|
| `encoding_extension` | *required* | The encoding extension ID to use for decoding traceV3 files (must be "macosunifiedlogencoding") |
| `tracev3_paths` | `[]` | Alternate paths to TraceV3 files. Overrides default include patterns when specified |
| `include` | `["/var/db/diagnostics/"]` | File patterns to include (inherited from fileconsumer) |
| `exclude` | `[]` | List of file patterns to exclude (inherited from fileconsumer) |
| `start_at` | `end` | Whether to read from `beginning` or `end` of existing files (inherited from fileconsumer) |
| `poll_interval` | `200ms` | How often to poll for new files (inherited from fileconsumer) |
| `max_log_size` | `1MiB` | Maximum size of a single log entry (inherited from fileconsumer) |
| `max_concurrent_files` | `1024` | Maximum number of files to read concurrently (inherited from fileconsumer) |

### File Consumer Configuration

This receiver inherits all configuration options from the Stanza file consumer. See the [fileconsumer documentation](../../pkg/stanza/fileconsumer/README.md) for additional options.

## Prerequisites

1. **Encoding Extension**: You must configure the `macosunifiedlogencoding` encoding extension (currently under development).

2. **File Access**: The collector must have read access to the traceV3 files, typically located in:
   - `/var/db/diagnostics/Persist/` - Persistent logs
   - `/var/db/diagnostics/Special/` - Special logs
   - `/var/db/diagnostics/signpost/` - Signpost logs

3. **macOS System**: This receiver is designed specifically for macOS systems.

## Example Use Cases

### Real-time System Log Monitoring

```yaml
receivers:
  macosunifiedloggingreceiver:
    encoding_extension: "macosunifiedlogencoding"
    include:
      - "/var/db/diagnostics/Persist/*.tracev3"
    start_at: end
    poll_interval: 100ms
```

### Historical Log Analysis

```yaml
receivers:
  macosunifiedloggingreceiver:
    encoding_extension: "macosunifiedlogencoding"
    tracev3_paths: 
      - "/path/to/archived/logs/*.tracev3"
    start_at: beginning
```

### Custom Log Archive Location

```yaml
receivers:
  macosunifiedloggingreceiver:
    encoding_extension: "macosunifiedlogencoding"
    tracev3_paths:
      - "/Users/admin/exported_logs/*.tracev3"
    start_at: beginning
```

## Output Format

Once implemented, the receiver will output OpenTelemetry logs with the following resource attributes:

- `log.file.path`: Path to the source traceV3 file
- `log.file.name`: Name of the source traceV3 file

Log records will include:
- Timestamp (converted from Mach absolute time)
- Message content
- Log level (Default, Info, Debug, Error, Fault)
- Process information (PID, effective UID)
- Thread and activity IDs
- Subsystem and category information
- Event type (Log, Signpost, Activity, etc.)

## Current Limitations

- **Encoding extension not implemented**: The receiver framework is ready but cannot decode traceV3 files yet
- Only supports traceV3 format (macOS 10.12+)
- Requires appropriate file system permissions
- Binary file format parsing depends on the encoding extension implementation
- Performance may vary with large traceV3 files

## Development Status

This receiver is currently under active development. The main components implemented:

- ✅ Receiver factory and configuration
- ✅ File consumer integration using Stanza
- ✅ Basic receiver framework
- ❌ Encoding extension for traceV3 files (in development)

## Troubleshooting

### Common Issues

1. **Permission Denied**: Ensure the collector has read access to traceV3 files
2. **Encoding Extension Not Found**: Verify the encoding extension is configured and the ID matches
3. **No Logs Received**: Check file patterns and ensure traceV3 files exist in the specified paths
4. **Parse Errors**: Verify the encoding extension supports the traceV3 file version
