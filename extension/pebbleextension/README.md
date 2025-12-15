# Pebble Extension

The Pebble Extension provides persistent storage for OpenTelemetry Collector components using [Pebble](https://github.com/cockroachdb/pebble), a high-performance embedded key-value database written in Go. Each component receives an isolated Pebble instance for data isolation and reliability.

## Configuration

| Field              | Type     | Default | Required | Description                                                                                       |
| ------------------ | -------- | ------- | -------- | ------------------------------------------------------------------------------------------------- |
| directory.path     | string   |         | `true`   | Directory path where Pebble database files will be stored. Each component gets a subdirectory.    |
| cache.size         | int64    | `0`     | `false`  | Size in bytes of the block cache. When 0, uses Pebble's default cache behavior. Larger values improve read performance at the cost of memory usage. |
| sync               | bool     | `false` | `false`  | Whether to sync writes to disk immediately. `false` provides better performance while still being durable. |

## Example Configuration

### Basic Configuration

```yaml
extensions:
  pebble:
    directory:
      path: /var/lib/otelcol/pebble

processors:
  batch:
    storage: pebble

exporters:
  otlp:
    endpoint: otelcol:4317

service:
  extensions: [pebble]
  pipelines:
    logs:
      receivers: [otlp]
      processors: [batch]
      exporters: [otlp]
```

## Advanced Configuration

This extension uses Pebble's defaults, which are optimized for most production workloads. The defaults provide a balance of performance, memory usage, and durability suitable for telemetry storage.

For advanced use cases, the following configuration options are exposed:

- **cache.size**: Configures the block cache size. Larger caches improve read performance at the cost of memory usage. Recommended when running on systems with available memory or when read performance is critical.
- **sync**: Controls write durability. The default (`false`) provides optimal performance for most scenarios while maintaining durability guarantees through write-ahead logging.

Refer to the [Pebble documentation](https://github.com/cockroachdb/pebble) for detailed information about the database.

Full configuration example:

```yaml
extensions:
  pebble:
    directory:
      path: /var/lib/otelcol/pebble
    cache:
      size: 536870912  # 512MB cache
    sync: false

processors:
  batch:
    storage: pebble

exporters:
  otlp:
    endpoint: otelcol:4317

service:
  extensions: [pebble]
  pipelines:
    logs:
      receivers: [otlp]
      processors: [batch]
      exporters: [otlp]

```

## Component Isolation

Each component that uses the pebble extension gets an isolated database instance at:

```
{directory.path}/{kind}_{type}_{component_name}_{name}/
```

For example, with `directory.path: $OIQ_OTEL_COLLECTOR_STORAGE`:

```
$OIQ_OTEL_COLLECTOR_STORAGE/processor_batch_default/
$OIQ_OTEL_COLLECTOR_STORAGE/exporter_otlp_backup/
```

This ensures components cannot interfere with each other's data.
