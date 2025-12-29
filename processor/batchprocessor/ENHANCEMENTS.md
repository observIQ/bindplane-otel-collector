# Batch Processor Enhancements

This document describes two new enhancements to the OpenTelemetry Collector batch processor:

1. **Byte-based batching** - Trigger batch sends based on byte size
2. **Attribute-based grouping** - Group telemetry data by attribute values

## Enhancement 1: Byte-based Batching

### Overview
The batch processor can now trigger batch sends based on the accumulated byte size of the data, in addition to the existing item count and timeout triggers.

### Configuration Options

- **`send_batch_size_bytes`** (default = 0): Size of a batch in bytes which after hit, will trigger it to be sent. When set to zero, the batch size in bytes is ignored. This works in conjunction with `send_batch_size` - a batch is sent when **either** threshold is reached.

- **`send_batch_max_size_bytes`** (default = 0): The maximum size of a batch in bytes. It must be larger than or equal to `send_batch_size_bytes`. Larger batches are split into smaller units. Default value of 0 means no maximum size in bytes.

### Example Configuration

```yaml
processors:
  batch:
    # Batch based on byte size
    send_batch_size_bytes: 1048576  # 1MB
    send_batch_max_size_bytes: 10485760  # 10MB max
    timeout: 10s
```

```yaml
processors:
  batch:
    # Use both item count and byte size thresholds
    # Whichever is reached first will trigger the send
    send_batch_size: 8192  # 8192 items
    send_batch_size_bytes: 5242880  # 5MB
    timeout: 10s
```

### Use Cases

- **Network bandwidth management**: Control outbound data size to prevent network saturation
- **Backend limitations**: Respect downstream system payload size limits
- **Cost optimization**: Optimize for pricing based on data volume
- **Memory management**: Prevent excessive memory usage with large batches

## Enhancement 2: Attribute-based Grouping

### Overview
The batch processor can now group telemetry data by specific attribute values from the data itself, ensuring that data with the same attribute values is batched together.

### Configuration Options

- **`batch_group_by_attributes`** (default = []): A list of attribute keys used to form distinct batchers. If this setting is empty, data will be batched in FIFO order. When non-empty, one batcher will be used per distinct combination of values for the listed attribute keys.

  Attributes are extracted from the telemetry data itself:
  - For traces: span attributes, resource attributes, or scope attributes (checked in that order)
  - For logs: log record attributes, resource attributes, or scope attributes (checked in that order)
  - For metrics: datapoint attributes, resource attributes, or scope attributes (checked in that order)

  Empty values and missing attributes are treated as distinct cases.
  Entries are case-sensitive. Duplicated entries will trigger a validation error.

- **`batch_group_by_attribute_limit`** (default = 0): Indicates the maximum number of batcher instances that will be created through a distinct combination of `batch_group_by_attributes`. Setting this to 0 means no limit.

### Example Configuration

```yaml
processors:
  batch:
    # Group traces by service name
    batch_group_by_attributes:
      - service.name
    batch_group_by_attribute_limit: 100
    send_batch_size: 1000
    timeout: 10s
```

```yaml
processors:
  batch:
    # Group by multiple attributes (e.g., tenant and environment)
    batch_group_by_attributes:
      - tenant.id
      - deployment.environment
    batch_group_by_attribute_limit: 500
    send_batch_size: 5000
    timeout: 5s
```

### Use Cases

- **Multi-tenant systems**: Ensure data from different tenants is batched separately
- **Service isolation**: Keep data from different services in separate batches
- **Environment separation**: Separate staging, production, and development data
- **Custom grouping**: Any scenario where you want to maintain grouping by specific attributes

### Important Notes

1. **Cannot combine with metadata_keys**: You cannot use both `metadata_keys` and `batch_group_by_attributes` simultaneously. Choose one grouping strategy.

2. **Memory implications**: Each distinct attribute combination creates a new batcher instance with its own timer and pending batch. Set `batch_group_by_attribute_limit` appropriately to control memory usage.

3. **Cardinality**: Be mindful of the cardinality of your grouping attributes. High cardinality attributes (like request IDs or timestamps) can create many batchers and increase memory usage.

4. **Attribute precedence**: Attributes are checked in order: span/log/datapoint attributes first, then scope attributes, then resource attributes. The first match is used.

## Combining Both Enhancements

You can use both byte-based batching and attribute-based grouping together:

```yaml
processors:
  batch:
    # Group by service name
    batch_group_by_attributes:
      - service.name
    batch_group_by_attribute_limit: 50
    
    # Use both item and byte thresholds
    send_batch_size: 10000
    send_batch_size_bytes: 5242880  # 5MB
    send_batch_max_size_bytes: 10485760  # 10MB
    
    timeout: 10s
```

In this configuration:
- Data is first grouped by `service.name` attribute
- Within each group, batches are sent when either 10,000 items OR 5MB is accumulated
- Batches larger than 10MB are automatically split
- Batches are also sent every 10 seconds if thresholds aren't reached

## Validation Rules

The processor enforces the following validation rules:

1. `send_batch_max_size_bytes` must be >= `send_batch_size_bytes`
2. `send_batch_max_size` must be >= `send_batch_size` (existing rule)
3. No duplicate entries in `batch_group_by_attributes`
4. Cannot use both `metadata_keys` and `batch_group_by_attributes`

## Telemetry

A new trigger type `triggerBatchSizeBytes` has been added to track when batches are sent due to byte size thresholds. This is reported in the existing telemetry metrics.

## Testing

The enhancements include comprehensive test coverage:
- Byte-based batching validation and functionality tests
- Attribute extraction tests for all signal types
- Attribute-based grouping tests
- Combined feature tests
- Validation tests for configuration rules

Run tests with:
```bash
go test ./processor/batchprocessor/...
```
