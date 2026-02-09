# Pebble Extension

The Pebble Extension provides persistent storage for OpenTelemetry Collector components using [Pebble](https://github.com/cockroachdb/pebble), a high-performance embedded key-value database written in Go. Each component receives an isolated Pebble instance for data isolation and reliability.

## Configuration

| Field                             | Type     | Default  | Required | Description                                                                                                                                                                                              |
| --------------------------------- | -------- | -------- | -------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| directory.path                    | string   |          | Yes      | **(Required)** Directory path where Pebble database files will be stored. Each component will use a subdirectory within this directory.                                                                  |
| directory.path_prefix             | string   | "pebble" | No       | Optional prefix added to each component's subdirectory name. Useful to avoid naming collisions when multiple Pebble extensions share a parent directory. Set to empty string `""` to disable the prefix. |
| cache.size                        | int64    | 0        | No       | Size (in bytes) of Pebble’s block cache. If set to 0, the default Pebble cache settings are used. Increasing this value can improve read performance at the cost of memory usage.                        |
| sync                              | bool     | true     | No       | Whether to sync writes to disk after every write. `true` offers safer durability guarantees. Setting `false` may improve performance but increases the chance of data loss in the event of a crash.      |
| compaction.interval               | duration | 30m      | No       | How often background compaction runs (e.g., "30m" for 30 minutes). Compaction reclaims space from deleted entries. Set to 0s to disable background compaction.                                           |
| compaction.compaction_concurrency | uint64   | 3        | No       | Number of concurrent background compaction jobs permitted. Increase for higher compaction throughput or decrease for reduced interference with other workloads.                                          |
| close_timeout                     | duration | 10s      | No       | Maximum time to wait for in-flight async operations to complete during shutdown. After this timeout, Close returns an error.                                                                           |

## Example Configuration

### Basic Configuration

```yaml
extensions:
  pebble:
    directory:
      path: /var/lib/otelcol/pebble

exporters:
  otlp:
    endpoint: otelcol:4317

service:
  extensions: [pebble]
  pipelines:
    logs:
      receivers: [otlp]
      exporters: [otlp]
```

## Advanced Configuration

This extension uses Pebble's defaults, which are optimized for most production workloads. The defaults provide a balance of performance, memory usage, and durability suitable for telemetry storage.

For advanced use cases, the following configuration options are exposed:

- **cache.size**: Configures the block cache size. Larger caches improve read performance at the cost of memory usage. Recommended when running on systems with available memory or when read performance is critical.
- **sync**: Controls write durability. The default (`true`) provides safer durability guarantees by syncing writes to disk immediately.
- **compaction.interval**: Controls how often background compaction checks run (default: 30 minutes). Set to 0 to disable background compaction.
- **compaction.concurrency**: Controls how many folders will be attempted to be compacted at the given interval (default: `3`). This parameter helps avoid causing way too many resources being used for compaction.
- **close_timeout**: Maximum time to wait for in-flight async operations to complete during shutdown (default: 10s). If async operations do not complete within this window, Close returns an error.

Refer to the [Pebble documentation](https://github.com/cockroachdb/pebble) for detailed information about the database.

Full configuration example:

```yaml
extensions:
  pebble:
    directory:
      path: /var/lib/otelcol/pebble
    cache:
      size: 536870912  # 512MB cache
    sync: true
    compaction:
      interval: 30m
      concurrency: 3
    close_timeout: 10s
exporters:
  otlp:
    endpoint: otelcol:4317

service:
  extensions: [pebble]
  pipelines:
    logs:
      receivers: [otlp]
      exporters: [otlp]

```

## Component Isolation

Each component that uses the pebble extension gets an isolated database instance at:

```
{directory.path}/{path_prefix}_{kind}_{type}_{component_name}_{name}/
```

When `directory.path_prefix` is not set or is empty, the format becomes:

```
{directory.path}/{kind}_{type}_{component_name}_{name}/
```

For example, with `directory.path: $OIQ_OTEL_COLLECTOR_STORAGE` and default prefix:

```
$OIQ_OTEL_COLLECTOR_STORAGE/pebble_processor_batch_default/
$OIQ_OTEL_COLLECTOR_STORAGE/pebble_exporter_otlp_backup/
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
