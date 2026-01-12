# Badger Extension

The Badger Extension provides persistent storage for OpenTelemetry Collector components using [BadgerDB](https://github.com/dgraph-io/badger), a fast embeddable key-value database written in Go. Each component receives an isolated BadgerDB instance for data isolation and reliability.

## Configuration

| Field                                 | Type     | Default           | Required | Description                                                                                  |
| ------------------------------------- | -------- | ----------------- | -------- | -------------------------------------------------------------------------------------------- |
| directory.path                        | string   |                   | `true`   | Directory path where BadgerDB files will be stored. Each component gets a subdirectory.      |
| directory.path_prefix                 | string   | `badger`          | `false`  | Optional prefix added to component directory names. Prevents naming collisions when multiple storage extensions share the same directory. Set to empty string to disable. |
| sync_writes                           | bool     | `true`           | `false`  | Whether to sync writes to disk immediately. `false` survives process crashes via mmap.       |
| memory.table_size                     | int64    | 67108864 (64MB)   | `false`  | Size of each memtable in bytes. Larger values improve write performance but use more memory. |
| memory.block_cache_size               | int64    | 268435456 (256MB) | `false`  | Size of block cache in bytes. Larger values improve read performance but use more memory.    |
| memory.value_log_file_size            | int64    | 67108864 (64MB)   | `false`  | Maximum size of each value log file in bytes. Smaller values reduce baseline disk usage but may increase file count. BadgerDB default is 1GB which is too large for queue workloads. |
| blob_garbage_collection.interval      | duration | 5m                | `false`  | Interval at which garbage collection runs on value logs. Set to 0 to disable.                |
| blob_garbage_collection.discard_ratio | float    | 0.5               | `false`  | Fraction of invalid data in a value log file to trigger GC. Must be between 0 and 1.         |
| compaction.numCompactors              | int      | 8                 | `false`  | Number of background compaction workers. Higher values = more aggressive LSM tree cleanup but increased CPU usage. Critical for queue workloads with heavy deletes. |
| compaction.numLevelZeroTables         | int      | 3                 | `false`  | Number of L0 tables that triggers LSM compaction. Lower values = earlier compaction, better disk cleanup. For queue workloads, consider reducing to 2 to prevent LSM bloat. |
| compaction.numLevelZeroTablesStall    | int      | 8                 | `false`  | Number of L0 tables that stalls writes. Must be >= numLevelZeroTables. Prevents unbounded L0 growth.         |
| telemetry.enabled                     | bool     | `false`           | `false`  | Whether to enable telemetry collection for the badger extension.                             |
| telemetry.update_interval             | duration | `1m`              | `false`  | The interval at which to update the telemetry metrics.                                       |

## Example Configuration

### Basic Configuration

```yaml
extensions:
  badger:
    directory:
      path: /var/lib/otelcol/badger

processors:
  batch:
    storage: badger

exporters:
  otlp:
    endpoint: otelcol:4317

service:
  extensions: [badger]
  pipelines:
    logs:
      receivers: [otlp]
      processors: [batch]
      exporters: [otlp]
```

## Advanced Configuration

This extension uses BadgerDB's defaults, which are optimized for most production workloads. The defaults provide a balance of performance, memory usage, and durability suitable for telemetry storage.

For advanced use cases, the following configuration levers are exposed:

- **memory.table_size** and **memory.block_cache_size**: Control memory allocation. Only adjust if you have specific memory constraints or performance requirements.
- **memory.value_log_file_size**: Controls the maximum size of value log files. The default (64MB) is optimized for queue workloads. BadgerDB's default is 1GB which causes high baseline disk usage.
- **blob_garbage_collection**: Controls disk space reclamation. The defaults (5m interval, 0.5 discard ratio) work well for most scenarios.
- **compaction**: Controls LSM tree cleanup. This is critical for queue workloads that perform heavy deletes. The LSM tree can accumulate tombstones (deleted entries) that consume disk space. Use these settings to trigger compaction earlier and more aggressively:
  - **num_level_zero_tables**: Lower values trigger compaction sooner, preventing LSM bloat. For queue workloads with high delete rates, consider reducing from default 3 to 2.
  - **num_compactors**: More workers clean up tombstones faster but use more CPU. For queue workloads struggling with disk usage, may need to increase from default 8 to 12+.
- **sync_writes**: Controls write durability. The default (`false`) provides good performance while surviving process crashes via memory-mapped writes.

Refer to the [BadgerDB documentation](https://github.com/dgraph-io/badger) for detailed information about these settings.

### Full Configuration Example

```yaml
extensions:
  badger:
    directory:
      path: /var/lib/otelcol/badger
    sync_writes: true
    memory:
      table_size: 134217728        # 128MB
      block_cache_size: 536870912  # 512MB
      value_log_file_size: 67108864  # 64MB
    compaction:
      num_compactors: 12            # Number of compaction workers (default: 12)
      num_level_zero_tables: 2      # Start compaction when 2 L0 tables (default: 2)
      num_level_zero_tables_stall: 12  # Stall writes at 12 L0 tables (default: 0=disabled)
    blob_garbage_collection:
      interval: 10m
      discard_ratio: 0.6
    telemetry:
      enabled: true
      update_interval: 30s

exporters:
  otlp:
    endpoint: otelcol:4317

service:
  extensions: [badger]
  pipelines:
    logs:
      receivers: [otlp]
      exporters: [otlp]

```

### Queue Workload Configuration (Optimized for Disk Usage)

For queue workloads with high throughput and heavy deletes (like OTel persistent queues), the LSM tree can accumulate tombstones that consume significant disk space. This configuration is optimized to minimize LSM disk bloat:

```yaml
extensions:
  badger:
    directory:
      path: /var/lib/otelcol/badger
    sync_writes: false          # Rely on mmap for durability
    memory:
      table_size: 67108864       # 64MB (default)
      block_cache_size: 268435456  # 256MB (default)
      value_log_file_size: 67108864  # 64MB (reduces baseline disk usage)
    blob_garbage_collection:
      interval: 5m
      discard_ratio: 0.5
    compaction:
      numCompactors: 8           # Default; increase to 12-16 if LSM usage still high
      numLevelZeroTables: 2      # Reduced from 3 to trigger compaction sooner
      numLevelZeroTablesStall: 8 # Prevents writes from blocking
    telemetry:
      enabled: true
      update_interval: 30s

processors:
  batch:
    storage: badger

exporters:
  otlp:
    endpoint: backend:4317

service:
  extensions: [badger]
  pipelines:
    logs:
      receivers: [otlp]
      processors: [batch]
      exporters: [otlp]

```

**Key differences for queue workloads:**
- `numLevelZeroTables: 2`: Triggers LSM compaction earlier to prevent tombstone accumulation
- Enable telemetry to monitor `LSMUsage` vs `ValueLogUsage` separately
- If LSM usage remains high after this change, increase `numCompactors` to 12+ (trades CPU for better disk cleanup)

## Component Isolation

Each component that uses the badger extension gets an isolated database instance at:

```
{directory.path}/{path_prefix}_{kind}_{type}_{component_name}_{name}/
```

When `directory.path_prefix` is not set or is empty, the format becomes:

```
{directory.path}/{kind}_{type}_{component_name}_{name}/
```

For example, with `directory.path: $OIQ_OTEL_COLLECTOR_STORAGE` and default prefix:

```
$OIQ_OTEL_COLLECTOR_STORAGE/badger_processor_batch_default/
$OIQ_OTEL_COLLECTOR_STORAGE/badger_exporter_otlp_backup/
```

When sharing a storage directory with other extensions, the prefix prevents naming collisions:

```yaml
extensions:
  badger:
    directory:
      path: /var/lib/otelcol/storage
      path_prefix: badger
  pebble:
    directory:
      path: /var/lib/otelcol/storage
      path_prefix: pebble
```

This ensures components cannot interfere with each other's data, even when using multiple storage backends in the same directory.
