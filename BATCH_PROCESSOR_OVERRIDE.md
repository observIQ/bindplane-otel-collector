# Enhanced Batch Processor in bindplane-otel-collector

This document explains the enhanced batch processor located at `processor/batchprocessor/` in this repository.

## What's Different?

The batch processor now includes two major enhancements:

1. **Byte-based batching** - Trigger batch sends based on payload size in bytes
2. **Attribute-based grouping** - Group telemetry data by specific attribute values

## How It Works

The integration uses Go's `replace` directive in `go.mod` to override the standard batch processor with our enhanced local version:

```go
// In go.mod (at the end)
replace go.opentelemetry.io/collector/processor/batchprocessor => ./processor/batchprocessor
```

This tells Go to use the enhanced batch processor from `processor/batchprocessor/` instead of downloading it from the official OpenTelemetry repository.

## Building

1. **The enhanced processor is already in this repo** at `processor/batchprocessor/`

2. **Build the collector**:
   ```bash
   cd /home/dj/repos/bindplane-otel-collector
   go build -o observiq-collector ./cmd/collector/
   ```

3. **Run with your config**:
   ```bash
   ./observiq-collector --config your-config.yaml
   ```

## Configuration Examples

### Example 1: Byte-based Batching

```yaml
processors:
  batch:
    send_batch_size_bytes: 1048576      # 1MB trigger
    send_batch_max_size_bytes: 5242880  # 5MB max
    timeout: 10s
```

### Example 2: Attribute-based Grouping

```yaml
processors:
  batch:
    batch_group_by_attributes:
      - service.name
      - tenant.id
    batch_group_by_attribute_limit: 100
    send_batch_size: 1000
    timeout: 5s
```

### Example 3: Combined Features

```yaml
processors:
  batch:
    # Group by attributes
    batch_group_by_attributes:
      - service.name
      - tenant.id
    batch_group_by_attribute_limit: 200
    
    # Use both byte and item thresholds (whichever is reached first)
    send_batch_size: 5000
    send_batch_size_bytes: 2097152      # 2MB
    send_batch_max_size_bytes: 10485760  # 10MB
    
    timeout: 10s
```

## Testing Your Configuration

A test configuration file is provided at `test-batch-enhancements.yaml`. To test:

```bash
./observiq-collector --config test-batch-enhancements.yaml
```

This configuration includes three different batch processor instances:
- `batch/bytes` - Byte-based batching only
- `batch/by_service` - Attribute-based grouping only
- `batch/combined` - Both features combined

## Updating the Batch Processor

If you make changes to the batch processor code in `processor/batchprocessor/`:

1. **Edit the files** in `processor/batchprocessor/`

2. **Rebuild**:
   ```bash
   cd /home/dj/repos/bindplane-otel-collector
   make build
   # or
   go build -o observiq-collector ./cmd/collector/
   ```

3. **Changes take effect immediately** in the new build

## Verifying the Integration

To verify that your modified batch processor is being used:

1. **Check the go.mod file**:
   ```bash
   grep "batchprocessor =>" go.mod
   ```
   Should show:
   ```
   replace go.opentelemetry.io/collector/processor/batchprocessor => ./processor/batchprocessor
   ```

2. **Run the collector with a config using new features**:
   ```bash
   ./observiq-collector --config test-batch-enhancements.yaml
   ```
   
3. **Look for successful startup** - No errors about unknown config fields

## Reverting to Standard Batch Processor

To use the standard batch processor instead:

1. **Remove the replace directive**:
   ```bash
   # Edit go.mod and remove these lines:
   # // Use local modified batch processor with byte-based and attribute-based batching enhancements
   # replace go.opentelemetry.io/collector/processor/batchprocessor => ./processor/batchprocessor
   ```
   
2. **Remove the local processor directory** (optional):
   ```bash
   rm -rf processor/batchprocessor
   ```

3. **Update dependencies**:
   ```bash
   go mod tidy
   ```

4. **Rebuild**:
   ```bash
   make build
   ```

## New Configuration Options

### Byte-based Batching

- **`send_batch_size_bytes`** (uint64, default: 0)
  - Byte size threshold to trigger batch send
  - Works with `send_batch_size` - either threshold can trigger send
  - 0 = disabled

- **`send_batch_max_size_bytes`** (uint64, default: 0)
  - Maximum batch size in bytes before splitting
  - Must be >= `send_batch_size_bytes`
  - 0 = no limit

### Attribute-based Grouping

- **`batch_group_by_attributes`** ([]string, default: [])
  - List of attribute keys to group by
  - Attributes extracted from telemetry data (span/log/metric attributes, then scope, then resource)
  - Cannot be used with `metadata_keys`

- **`batch_group_by_attribute_limit`** (uint32, default: 0)
  - Maximum number of distinct attribute combinations
  - Controls memory usage
  - 0 = no limit

## Troubleshooting

### Build Errors

**Error**: `module not found`
**Solution**: Ensure the directory structure is correct and the path in the replace directive matches

**Error**: `validation failed`
**Solution**: Check your configuration syntax - the new fields are case-sensitive

### Runtime Issues

**Issue**: Collector ignores new config options
**Solution**: Verify the replace directive is present and rebuild

**Issue**: Out of memory
**Solution**: Set `batch_group_by_attribute_limit` to limit the number of batchers

## Documentation

For detailed documentation on the enhancements, see:
- `processor/batchprocessor/ENHANCEMENTS.md` - Feature documentation
- `processor/batchprocessor/COMBINED_USAGE.md` - Combined usage guide
- `processor/batchprocessor/example-combined-config.yaml` - More examples

## Support

The enhanced batch processor includes comprehensive tests:
```bash
cd processor/batchprocessor
go test -v
```

All tests should pass, including:
- Byte-based batching tests
- Attribute-based grouping tests
- Combined feature tests
- Original batch processor tests (backward compatibility)
