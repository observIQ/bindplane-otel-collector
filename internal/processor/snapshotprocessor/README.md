# Snapshot Processor

Supported pipelines: logs, metrics, traces

This processor saves OTLP payloads into snapshots that can be reported to [Bindplane](https://bindplane.com/).

This processor is only used by bindplane-agent. To add snapshot support to an agent built with OpenTelemetry Collector Builder, use the non-internal [snapshotprocessor](/processor/snapshotprocessor/README.md) instead.

## Configuration

The following options may be configured:
- `enabled` (default: true): When `true` signals that snapshots are being taken of data passing through this processor. If false this processor acts as a no-op.

### Example configuration

```yaml
processors:
  snapshot:
    enabled: true
```

