# Azure Blob Storage Polling Receiver

Continuously polls Azure Blob Storage at configurable intervals and dynamically adjusts the time window to collect only new data from each interval. This receiver is designed for ongoing data collection from Azure Blob Storage that was stored using the Azure Blob Exporter [../../exporter/azureblobexporter/README.md].

## Important Note

Unlike the `azureblobrehydrationreceiver` which is a one-time rehydration receiver, this receiver operates continuously. It polls Azure Blob Storage at regular intervals and automatically adjusts the time window to collect only new blobs since the last poll, making it suitable for production monitoring scenarios.

## Comparison with Azure Blob Rehydration Receiver

| Feature                 | azureblobpollingreceiver         | azureblobrehydrationreceiver      |
| ----------------------- | -------------------------------- | --------------------------------- |
| Operation Mode          | Continuous polling               | One-time rehydration              |
| Time Range              | Dynamic (based on poll interval) | Static (user-specified)           |
| Use Case                | Production monitoring            | Historical data recovery          |
| Configuration           | `poll_interval`                  | `starting_time` and `ending_time` |
| Stops After Empty Polls | No                               | Yes (after 3 consecutive)         |

## Minimum Agent Versions

- Introduced: v1.92.0

## Supported Pipelines

- Metrics
- Logs
- Traces

## How it works

1. On startup, the receiver loads the checkpoint from storage (if configured) to determine where it left off.
2. The receiver immediately runs the first poll, looking back by `initial_lookback` duration (defaults to `poll_interval`).
3. For each subsequent poll at the `poll_interval`:
   - The receiver calculates a dynamic time window from the last poll time to now
   - It streams blobs from Azure Blob Storage in the specified container
   - Each blob path is parsed to extract the timestamp and telemetry type
   - Blobs within the time window and matching the receiver's telemetry type are downloaded and processed
   - The checkpoint is updated with the current poll time and processed blobs
4. The cycle repeats continuously until the collector is shut down.

### Dynamic Time Windows

The receiver automatically manages time windows:

- **First Poll**: Uses `initial_lookback` to determine how far back to look (e.g., if `initial_lookback: 1h`, it will process blobs from the last hour)
- **Subsequent Polls**: Uses the timestamp of the last successful poll as the start time, and the current time as the end time
- **After Restart**: If a checkpoint exists, resumes from the last poll time; otherwise, uses `initial_lookback` again

### Checkpoint Management

The receiver uses a checkpoint to track:

- `LastPollTime`: The timestamp when the last poll completed successfully
- `LastTs`: The timestamp from the last processed blob's path
- `ParsedEntities`: A set of blob names already processed in the current time bucket

This prevents duplicate processing of blobs and ensures data continuity across collector restarts.

## Configuration

| Field             | Type     | Default               | Required | Description                                                                                                                                                                 |
| ----------------- | -------- | --------------------- | -------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| connection_string | string   |                       | `true`   | The connection string to the Azure Blob Storage account. Can be found under the `Access keys` section of your storage account.                                              |
| container         | string   |                       | `true`   | The name of the container to poll from.                                                                                                                                     |
| poll_interval     | duration |                       | `true`   | The interval at which to poll for new blobs. Must be at least 1 minute. The receiver will continuously poll at this interval and collect blobs created since the last poll. |
| root_folder       | string   |                       | `false`  | The root folder that prefixes the blob path. Should match the `root_folder` value of the Azure Blob Exporter.                                                               |
| initial_lookback  | duration | same as poll_interval | `false`  | The duration to look back on the first poll when no checkpoint exists. For example, if set to `1h`, on first startup the receiver will look for blobs from the last hour.   |
| delete_on_read    | bool     | `false`               | `false`  | If `true` the blob will be deleted after being processed.                                                                                                                   |
| storage           | string   |                       | `false`  | The component ID of a storage extension. The storage extension persists checkpoint data across collector restarts, ensuring no data loss or duplication.                    |
| batch_size        | int      | `30`                  | `false`  | The number of blobs to download and process in the pipeline simultaneously. This parameter directly impacts performance by controlling the concurrent blob download limit.  |
| page_size         | int      | `1000`                | `false`  | The maximum number of blob information to request in a single API call.                                                                                                     |

## Example Configuration

### Basic Configuration

This configuration polls every 10 minutes for new blobs in the container `my-container`. On first startup, it will look back 10 minutes (the default `initial_lookback`).

```yaml
azureblobpolling:
  connection_string: "DefaultEndpointsProtocol=https;AccountName=storage_account_name;AccountKey=storage_account_key;EndpointSuffix=core.windows.net"
  container: "my-container"
  poll_interval: 10m
  batch_size: 100
  page_size: 1000
```

### Custom Initial Lookback Configuration

This configuration polls every 5 minutes but looks back 1 hour on the first poll. This is useful when starting the receiver for the first time and you want to collect more historical data.

```yaml
azureblobpolling:
  connection_string: "DefaultEndpointsProtocol=https;AccountName=storage_account_name;AccountKey=storage_account_key;EndpointSuffix=core.windows.net"
  container: "my-container"
  poll_interval: 5m
  initial_lookback: 1h
  batch_size: 100
  page_size: 1000
```

### Using Storage Extension Configuration

This configuration shows using a storage extension to persist checkpoint data across agent restarts. The `storage` field is set to the component ID of the storage extension. This is **highly recommended** for production use to prevent data loss or duplication.

```yaml
extensions:
  file_storage:
    directory: $OIQ_OTEL_COLLECTOR_HOME/storage
receivers:
  azureblobpolling:
    connection_string: "DefaultEndpointsProtocol=https;AccountName=storage_account_name;AccountKey=storage_account_key;EndpointSuffix=core.windows.net"
    container: "my-container"
    poll_interval: 15m
    storage: "file_storage"
    batch_size: 100
    page_size: 1000
```

### Root Folder Configuration

This configuration specifies an additional field `root_folder` to match the `root_folder` value of the Azure Blob Exporter. The `root_folder` value in the exporter will prefix the blob path with the root folder and it needs to be accounted for in the polling receiver.

Such a path could look like the following:

```
root/year=2023/month=10/day=01/hour=13/minute=30/metrics_12345.json
root/year=2023/month=10/day=01/hour=13/minute=30/logs_12345.json
root/year=2023/month=10/day=01/hour=13/minute=30/traces_12345.json
```

```yaml
azureblobpolling:
  connection_string: "DefaultEndpointsProtocol=https;AccountName=storage_account_name;AccountKey=storage_account_key;EndpointSuffix=core.windows.net"
  container: "my-container"
  poll_interval: 10m
  root_folder: "root"
  batch_size: 100
  page_size: 1000
```

### Delete on Read Configuration

This configuration enables the `delete_on_read` functionality which will delete a blob from Azure after it has been successfully processed into OTLP data and sent to the next component in the pipeline. **Use with caution** as this permanently deletes data from Azure Blob Storage.

```yaml
azureblobpolling:
  connection_string: "DefaultEndpointsProtocol=https;AccountName=storage_account_name;AccountKey=storage_account_key;EndpointSuffix=core.windows.net"
  container: "my-container"
  poll_interval: 10m
  delete_on_read: true
  batch_size: 100
  page_size: 1000
```

### Complete Production Configuration

This example shows a complete production-ready configuration with storage extension for persistence and appropriate polling settings.

```yaml
extensions:
  file_storage:
    directory: $OIQ_OTEL_COLLECTOR_HOME/storage

receivers:
  azureblobpolling/metrics:
    connection_string: "DefaultEndpointsProtocol=https;AccountName=storage_account_name;AccountKey=storage_account_key;EndpointSuffix=core.windows.net"
    container: "otel-metrics"
    poll_interval: 5m
    initial_lookback: 30m
    storage: "file_storage"
    batch_size: 50
    page_size: 1000

  azureblobpolling/logs:
    connection_string: "DefaultEndpointsProtocol=https;AccountName=storage_account_name;AccountKey=storage_account_key;EndpointSuffix=core.windows.net"
    container: "otel-logs"
    poll_interval: 2m
    initial_lookback: 15m
    storage: "file_storage"
    batch_size: 100
    page_size: 1000

service:
  extensions: [file_storage]
  pipelines:
    metrics:
      receivers: [azureblobpolling/metrics]
      exporters: [otlp]
    logs:
      receivers: [azureblobpolling/logs]
      exporters: [otlp]
```

## Behavior on Restart

When the collector restarts:

1. **With Storage Extension**: The receiver loads the checkpoint and resumes polling from the `LastPollTime`. This ensures no data is missed or duplicated.
2. **Without Storage Extension**: The receiver starts fresh with no checkpoint, using `initial_lookback` to determine the starting point. This may result in duplicate processing of recent data.

For production deployments, **always use a storage extension** to maintain state across restarts.

## Performance Tuning

- **poll_interval**: Adjust based on your data ingestion rate. Shorter intervals provide lower latency but increase API calls.
- **batch_size**: Controls concurrent blob downloads. Higher values improve throughput but increase memory usage.
- **page_size**: Number of blobs retrieved per API call. Higher values reduce API calls but may increase latency.

Recommended starting values:

- High-frequency data (< 5 min between blobs): `poll_interval: 2m`, `batch_size: 100`
- Medium-frequency data (5-15 min): `poll_interval: 10m`, `batch_size: 50`
- Low-frequency data (> 15 min): `poll_interval: 30m`, `batch_size: 30`

## Flexible Time Patterns

The receiver supports three modes for extracting timestamps from blob paths:

### Mode 1: Structured Path (Default)

Uses the default folder structure with explicit labels. This is the recommended mode when using the Azure Blob Exporter.

**Expected blob path:**

```
year=2025/month=12/day=05/hour=14/minute=30/logs_12345.json
```

The filename must contain:

- `logs_` for logs pipeline
- `metrics_` for metrics pipeline
- `traces_` for traces pipeline

### Mode 2: LastModified Timestamp

Uses the blob's LastModified property instead of parsing the path. Useful for blobs from external sources that don't follow a specific naming structure.

```yaml
azureblobpolling:
  connection_string: "..."
  container: "unstructured-logs"
  poll_interval: 2m
  use_last_modified: true
  telemetry_type: "logs" # Required when using use_last_modified
```

**Works with any blob path:**

```
application.log
data/2025/file.json
logs/app-server.log
```

### Mode 3: Custom Time Pattern

Extracts timestamps from custom path structures using patterns.

#### Named Placeholders

Use named placeholders for easy-to-read patterns:

```yaml
azureblobpolling:
  connection_string: "..."
  container: "logs"
  poll_interval: 1m
  time_pattern: "{year}/{month}/{day}/{hour}/{minute}"
  telemetry_type: "logs"
```

**Matches paths like:** `2025/12/05/14/30/application.json`

**Available placeholders:**

- `{year}` - 4 digits (2025)
- `{month}` - 2 digits (01-12)
- `{day}` - 2 digits (01-31)
- `{hour}` - 2 digits (00-23)
- `{minute}` - 2 digits (00-59)
- `{second}` - 2 digits (00-59)

**More examples:**

```yaml
# With prefix and custom separators
time_pattern: "logs/{year}-{month}-{day}/{hour}"
# Matches: logs/2025-12-05/14/app.json

# No minutes
time_pattern: "{year}/{month}/{day}/{hour}"
# Matches: 2025/12/05/14/data.json
```

#### Go Time Format

Use Go's time format for more control:

```yaml
azureblobpolling:
  connection_string: "..."
  container: "metrics"
  poll_interval: 5m
  time_pattern: "2006/01/02/15/04"
  telemetry_type: "metrics"
```

**Go time format reference:**

- `2006` - year
- `01` - month
- `02` - day
- `15` - hour (24h format)
- `04` - minute
- `05` - second

**Example:** Pattern `"2006-01-02/15"` matches `2025-12-05/14/metrics.json`

## Filename Filtering

Use `filename_pattern` to filter blobs by their filename using regex. This is useful when multiple types of files are in the same container.

### Example: Filter Firewall Logs

```yaml
azureblobpolling:
  connection_string: "..."
  container: "security-logs"
  poll_interval: 1m
  time_pattern: "{year}/{month}/{day}/{hour}"
  telemetry_type: "logs"
  filename_pattern: "firewall\\d+_\\w+\\.json"
```

**Matches:** `firewall43_dfreds.json`, `firewall1_data.json`  
**Skips:** `application.json`, `system.log`

### Example: All JSON Files

```yaml
azureblobpolling:
  connection_string: "..."
  container: "data"
  poll_interval: 5m
  use_last_modified: true
  telemetry_type: "logs"
  filename_pattern: ".*\\.json"
```

**Matches:** Any file ending with `.json`  
**Skips:** `.log`, `.txt`, `.gz` files

### Example: Application Logs Only

```yaml
azureblobpolling:
  connection_string: "..."
  container: "logs"
  poll_interval: 2m
  time_pattern: "2006/01/02/15"
  telemetry_type: "logs"
  filename_pattern: "app-.*\\.(log|json)"
```

**Matches:** `app-server.log`, `app-client.json`, `app-api.log`  
**Skips:** `system.log`, `data.txt`

### Regex Tips

- Remember to escape special characters in YAML: `\d` becomes `\\d`
- `.` (dot) matches any character, use `\\.` to match a literal dot
- `*` means "zero or more of the previous", use `.*` to match any characters
- Test your regex at https://regex101.com/ before adding to config

## Advanced Configuration Examples

### Example: Mixed Data Sources with Filtering

```yaml
extensions:
  file_storage:
    directory: /var/lib/otelcol/storage

receivers:
  # Structured data from Azure Blob Exporter
  azureblobpolling/structured:
    connection_string: "..."
    container: "otel-data"
    poll_interval: 5m
    storage: "file_storage"

  # Unstructured application logs
  azureblobpolling/app-logs:
    connection_string: "..."
    container: "raw-logs"
    poll_interval: 2m
    root_folder: "production"
    time_pattern: "{year}/{month}/{day}/{hour}"
    telemetry_type: "logs"
    filename_pattern: "app-.*\\.json"
    storage: "file_storage"

  # External data using LastModified
  azureblobpolling/external:
    connection_string: "..."
    container: "third-party-logs"
    poll_interval: 10m
    use_last_modified: true
    telemetry_type: "logs"
    filename_pattern: ".*\\.(json|log)"
    storage: "file_storage"

service:
  extensions: [file_storage]
  pipelines:
    logs:
      receivers:
        - azureblobpolling/structured
        - azureblobpolling/app-logs
        - azureblobpolling/external
      exporters: [otlp]
```

## Additional Configuration Fields

| Field                      | Type   | Default | Required | Description                                                                                                                                                                                                                                       |
| -------------------------- | ------ | ------- | -------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| use_last_modified          | bool   | `false` | `false`  | When `true`, uses the blob's LastModified timestamp instead of parsing the folder structure. Can be combined with `time_pattern` when `use_time_pattern_as_prefix` is enabled.                                                                    |
| time_pattern               | string |         | `false`  | Custom pattern for extracting timestamps from blob paths. Supports named placeholders and Go time format. Can be combined with `use_last_modified` when `use_time_pattern_as_prefix` is enabled.                                                  |
| use_time_pattern_as_prefix | bool   | `false` | `false`  | When `true`, uses the `time_pattern` to generate efficient prefixes for Azure API calls, significantly reducing the number of blobs scanned. Requires `time_pattern` to be set. Can be combined with `use_last_modified` for optimal performance. |
| telemetry_type             | string |         | `false`  | Explicitly sets the telemetry type (`logs`, `metrics`, or `traces`). Required when using `time_pattern` or `use_last_modified`. Falls back to pipeline type if not set.                                                                           |
| filename_pattern           | string |         | `false`  | Regex pattern to filter blobs by filename. Only matching blobs are processed.                                                                                                                                                                     |

## Prefix Optimization for Large Containers

When dealing with containers that have a large number of blobs (100K+), the `use_time_pattern_as_prefix` option provides significant performance improvements by generating time-based prefixes for Azure API calls instead of listing all blobs.

### How It Works

Without prefix optimization, the receiver must list ALL blobs in the container and check each one's timestamp. With prefix optimization enabled, the receiver:

1. Parses the `time_pattern` to identify time components (year, month, day, hour)
2. Generates prefixes for each time bucket in the lookback window
3. Only lists blobs under those specific prefixes

For example, with `time_pattern: "{year}/{month}/{day}"` and a 5-minute lookback at 3:00 PM on January 27, 2025, the receiver generates the prefix `2025/01/27/` and only lists blobs under that path.

### Hybrid Mode: Prefix + LastModified

For blob structures that only include date (not time) in the path, combine `use_time_pattern_as_prefix` with `use_last_modified` for optimal performance:

- **Prefix optimization**: Efficiently lists only blobs from relevant date folders
- **LastModified filtering**: Precisely filters to blobs within the actual time window

```yaml
azureblobpolling:
  connection_string: "..."
  container: "logs"
  root_folder: "myapp/logs"
  poll_interval: 1m
  initial_lookback: 5m
  time_pattern: "{year}/{month}/{day}"
  use_time_pattern_as_prefix: true
  use_last_modified: true
  telemetry_type: "logs"
  storage: "file_storage"
```

**Why hybrid mode?** When the path only contains `{year}/{month}/{day}`, parsed timestamps resolve to midnight (00:00:00 UTC). A 5-minute lookback window at 3:00 PM would miss all blobs because midnight is outside the window. Using `use_last_modified: true` ensures precise time filtering based on the blob's actual modification time.

### Per-Category Receivers

For containers with multiple log categories (e.g., `logs/dns/`, `logs/firewall/`, `logs/app/`), create separate receivers for each category. This provides:

- More efficient prefix filtering per category
- Separate pipelines for different log types
- Better resource isolation and monitoring

```yaml
receivers:
  azureblobpolling/dns:
    connection_string: "..."
    container: "logs"
    root_folder: "logs/dns"
    poll_interval: 1m
    time_pattern: "{year}/{month}/{day}"
    use_time_pattern_as_prefix: true
    use_last_modified: true
    telemetry_type: "logs"
    storage: "file_storage"

  azureblobpolling/firewall:
    connection_string: "..."
    container: "logs"
    root_folder: "logs/firewall"
    poll_interval: 1m
    time_pattern: "{year}/{month}/{day}"
    use_time_pattern_as_prefix: true
    use_last_modified: true
    telemetry_type: "logs"
    storage: "file_storage"
```

## Performance Benchmarks and Recommendations

The following benchmarks were collected using Azure Blob Storage with containers containing 1M+ blobs.

### Test Environment

- Container: 1,000,000+ blobs across multiple categories
- Blob size: ~1 KB each (gzipped JSON logs, 10 events per blob)
- Poll interval: 1 minute
- Lookback: 5 minutes
- Configuration: batch_size=200, page_size=5000

### Benchmark Results

| Configuration                 | Poll Duration               | Notes                                                             |
| ----------------------------- | --------------------------- | ----------------------------------------------------------------- |
| `use_last_modified` only      | **7+ minutes** (incomplete) | Lists all 1M+ blobs every poll                                    |
| `time_pattern` only           | ~25 seconds                 | Fast listing, but timestamp parsing issues with day-only patterns |
| **Hybrid mode** (recommended) | **45-52 seconds**           | Prefix-optimized listing + precise LastModified filtering         |

### Processing Throughput

With the hybrid configuration processing new blobs:

- **Blob processing rate**: ~200 blobs/second
- **Event throughput**: ~2,000 events/second (at 10 events/blob)
- **Memory usage**: < 200 MB
- **CPU usage**: < 5% (I/O bound, not CPU bound)

### Configuration Recommendations

#### Small Containers (< 10,000 blobs)

Any configuration works well. Use the simplest option for your blob structure:

```yaml
azureblobpolling:
  connection_string: "..."
  container: "small-logs"
  poll_interval: 2m
  use_last_modified: true
  telemetry_type: "logs"
```

#### Medium Containers (10,000 - 100,000 blobs)

Consider using `time_pattern` for better performance:

```yaml
azureblobpolling:
  connection_string: "..."
  container: "medium-logs"
  poll_interval: 2m
  time_pattern: "{year}/{month}/{day}/{hour}"
  telemetry_type: "logs"
  batch_size: 100
```

#### Large Containers (100,000+ blobs)

**Required**: Use `use_time_pattern_as_prefix` to avoid listing all blobs:

```yaml
azureblobpolling:
  connection_string: "..."
  container: "large-logs"
  root_folder: "category/subcategory"
  poll_interval: 1m
  time_pattern: "{year}/{month}/{day}"
  use_time_pattern_as_prefix: true
  use_last_modified: true # Required for day-only patterns
  telemetry_type: "logs"
  batch_size: 200
  page_size: 5000
  storage: "file_storage"
```

#### Very Large Containers (1M+ blobs)

For containers with millions of blobs:

1. **Split by category**: Create separate receivers per log type/category
2. **Use hourly paths if possible**: `{year}/{month}/{day}/{hour}` provides 24x better filtering than daily paths
3. **Increase batch_size**: Use 200+ for better throughput
4. **Use storage extension**: Critical for checkpoint persistence

```yaml
# Example for 1M+ blob container with daily folders
azureblobpolling:
  connection_string: "..."
  container: "enterprise-logs"
  root_folder: "production/application"
  poll_interval: 1m
  initial_lookback: 24h # Cover full day for day-only patterns
  time_pattern: "{year}/{month}/{day}"
  use_time_pattern_as_prefix: true
  use_last_modified: true
  telemetry_type: "logs"
  batch_size: 200
  page_size: 5000
  storage: "file_storage"
```

### Blob Path Structure Recommendations

For optimal performance with large containers, structure blob paths with time components that support efficient prefix filtering:

| Structure                               | Prefix Efficiency | Recommendation                                           |
| --------------------------------------- | ----------------- | -------------------------------------------------------- |
| `{year}/{month}/{day}/{hour}/{minute}/` | Excellent         | Best for high-volume, low-latency                        |
| `{year}/{month}/{day}/{hour}/`          | Very Good         | Good balance of efficiency and simplicity                |
| `{year}/{month}/{day}/`                 | Good              | Use with hybrid mode (+ `use_last_modified`)             |
| `{category}/{year}/{month}/{day}/`      | Good              | Split receivers by category                              |
| Flat structure (no time in path)        | Poor              | Avoid for large containers; use `use_last_modified` only |

### Troubleshooting Performance Issues

**Poll times exceeding poll_interval:**

- Enable `use_time_pattern_as_prefix` if not already enabled
- Split into multiple receivers by category
- Increase `batch_size` for better throughput
- Consider adding hour to blob paths

**Missing blobs with day-only patterns:**

- Add `use_last_modified: true` to enable hybrid mode
- Verify `initial_lookback` covers at least 24 hours

**High memory usage:**

- Reduce `batch_size` to limit concurrent downloads
- Enable memory_limiter processor in the pipeline
