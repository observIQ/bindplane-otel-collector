# Using Attribute-Based Batching with Byte-Size Batching

## Yes! They Work Together Perfectly

The attribute-based batching and byte-size batching features are **fully compatible** and can be used together. Here's how it works:

## How It Works

When both features are enabled:

1. **Data is first grouped by attributes** - Each unique combination of attribute values gets its own batch queue
2. **Within each group, byte/item thresholds apply** - Each group's batch is sent when either threshold is reached
3. **Byte limits apply per group** - Each attribute combination respects the byte limits independently

## Architecture

```
Incoming Data
    ↓
Extract Attributes (e.g., service.name, tenant.id)
    ↓
Route to Appropriate Batcher
    ↓
├─ Batcher for service=A, tenant=1
│  ├─ Check: bytes >= send_batch_size_bytes? → SEND
│  ├─ Check: items >= send_batch_size? → SEND
│  └─ Check: timeout expired? → SEND
│
├─ Batcher for service=A, tenant=2
│  └─ (same checks)
│
└─ Batcher for service=B, tenant=1
   └─ (same checks)
```

## Common Use Cases

### 1. Multi-Tenant Systems with Bandwidth Control

**Scenario**: SaaS platform with multiple tenants, need to control bandwidth per tenant

```yaml
processors:
  batch:
    batch_group_by_attributes:
      - tenant.id
    batch_group_by_attribute_limit: 1000
    
    send_batch_size: 10000
    send_batch_size_bytes: 5242880      # 5MB per tenant
    send_batch_max_size_bytes: 10485760  # 10MB max
    timeout: 10s
```

**Benefit**: Each tenant's data is batched separately, and no tenant can send more than 5MB in a single batch.

### 2. Service-Based Batching with Memory Limits

**Scenario**: Microservices architecture, want to prevent any single service from overwhelming the collector

```yaml
processors:
  batch:
    batch_group_by_attributes:
      - service.name
    batch_group_by_attribute_limit: 100
    
    send_batch_size: 5000
    send_batch_size_bytes: 2097152  # 2MB
    timeout: 5s
```

**Benefit**: High-volume services can't monopolize resources; all services get fair batching.

### 3. Environment + Service Isolation with Strict Byte Limits

**Scenario**: Multiple environments (prod, staging, dev) with multiple services, need strict separation

```yaml
processors:
  batch:
    batch_group_by_attributes:
      - deployment.environment
      - service.name
    batch_group_by_attribute_limit: 500
    
    send_batch_size: 0  # No item limit
    send_batch_size_bytes: 1048576  # 1MB strict
    send_batch_max_size_bytes: 2097152  # 2MB max
    timeout: 15s
```

**Benefit**: Production data never mixes with staging, and byte limits are strictly enforced per service per environment.

## Configuration Guidelines

### When to Use Both Features

Use both when you need:
- ✅ Logical grouping of data (by tenant, service, environment, etc.)
- ✅ Bandwidth/payload size control
- ✅ Fair resource distribution across groups
- ✅ Compliance with downstream system limits

### Configuration Tips

1. **Set realistic byte limits**: Calculate based on your network bandwidth and backend limits
   ```yaml
   # Example: 10Mbps network, want max 1 batch/sec per group
   send_batch_size_bytes: 1250000  # 1.25MB (10Mbps / 8)
   ```

2. **Use both item and byte thresholds**: Provides flexibility for variable-sized data
   ```yaml
   send_batch_size: 10000          # Handles many small items
   send_batch_size_bytes: 5242880  # Handles few large items
   ```

3. **Set appropriate cardinality limits**: Based on memory constraints
   ```yaml
   # Each batcher uses ~1-10MB of memory depending on batch size
   # For 2GB available memory:
   batch_group_by_attribute_limit: 200  # Conservative
   ```

4. **Adjust timeout based on throughput**: Low-traffic groups need shorter timeouts
   ```yaml
   timeout: 10s  # For low-traffic scenarios
   timeout: 1s   # For high-traffic scenarios
   ```

## Memory Implications

When using both features together, memory usage is:

```
Total Memory ≈ (Number of Distinct Attribute Combinations) × (Average Batch Size)
```

Example:
- 50 distinct attribute combinations (services × tenants)
- 5MB average batch size
- **Total: ~250MB**

Use `batch_group_by_attribute_limit` to cap memory usage.

## Performance Characteristics

### Throughput
- ✅ **No performance penalty** - Attribute extraction is fast (< 1μs)
- ✅ **Parallel processing** - Each group processes independently
- ✅ **Lock-free reads** - Uses sync.Map for efficient concurrent access

### Latency
- **First batch**: Depends on thresholds and timeout
- **Subsequent batches**: Minimal latency, sent as soon as thresholds are met
- **Per-group fairness**: No group blocks others

## Real-World Example

A large SaaS platform configuration:

```yaml
processors:
  batch/production:
    # Group by customer and service
    batch_group_by_attributes:
      - customer.id
      - service.name
    batch_group_by_attribute_limit: 2000  # 1000 customers × 2 services avg
    
    # 5MB or 10k items, whichever first
    send_batch_size: 10000
    send_batch_size_bytes: 5242880
    
    # Hard limit at 20MB
    send_batch_max_size_bytes: 20971520
    
    # Send at least every 30 seconds
    timeout: 30s
```

**Results**:
- ✅ Each customer's data stays separate
- ✅ No customer can send >5MB in one batch
- ✅ Fair resource allocation across 2000 customer-service combinations
- ✅ Automatic batching when customers are quiet (30s timeout)

## Testing Your Configuration

Use the test utilities to verify your configuration works as expected:

```bash
# Run all batch processor tests
cd processor/batchprocessor
go test -v

# Run specific combined feature tests
go test -v -run TestAttributeBasedBatchingWithByteSizeThresholds
go test -v -run TestMultipleAttributesWithByteBatching
```

## Monitoring

Watch these metrics to ensure proper operation:
- `otelcol_processor_batch_metadata_cardinality` - Number of active batchers
- `processor_batch_batch_send_size` - Item count per batch
- `processor_batch_batch_send_size_bytes` - Byte size per batch
- `processor_batch_batch_size_trigger_send` - Batches sent due to item threshold
- `processor_batch_timeout_trigger_send` - Batches sent due to timeout

## Troubleshooting

### Too Many Batchers Created
**Symptom**: High memory usage, many small batches
**Solution**: Lower `batch_group_by_attribute_limit` or use fewer grouping attributes

### Batches Too Small
**Symptom**: Many tiny batches being sent
**Solution**: Increase `send_batch_size_bytes` and `timeout`

### Batches Too Large
**Symptom**: Backend rejecting payloads
**Solution**: Lower `send_batch_max_size_bytes`

### Data Not Grouping Correctly
**Symptom**: Expected groups not forming
**Solution**: Verify attribute names are correct and present in the data. Check attribute precedence (span → scope → resource).

## Summary

✅ **Yes, you can use attribute-based batching with byte-size batching!**

The features complement each other perfectly:
- **Attribute batching** provides logical grouping
- **Byte batching** provides size control
- **Together** they give you precise control over how your data is batched and sent

See `example-combined-config.yaml` for complete configuration examples.
