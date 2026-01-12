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
| blob_garbage_collection.interval      | duration | 5m                | `false`  | Interval at which garbage collection runs on value logs. Set to 0 to disable.                |
| blob_garbage_collection.discard_ratio | float    | 0.5               | `false`  | Fraction of invalid data in a value log file to trigger GC. Must be between 0 and 1.         |
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
- **blob_garbage_collection**: Controls disk space reclamation. The defaults (5m interval, 0.5 discard ratio) work well for most scenarios.
- **sync_writes**: Controls write durability. The default (`false`) provides good performance while surviving process crashes via memory-mapped writes.

Refer to the [BadgerDB documentation](https://github.com/dgraph-io/badger) for detailed information about these settings.

Full configuration example:

```yaml
extensions:
  badger:
    directory:
      path: /var/lib/otelcol/badger
    sync_writes: true
    memory:
      table_size: 134217728        # 128MB
      block_cache_size: 536870912  # 512MB
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
