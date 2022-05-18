# Resource Attribute Transposer Processor

Supported pipelines: logs, metrics

This processor copies a resource level attribute to all individual logs or metric data points associated with the resource.
If they key already exists, no action is taken (the data points' attribute _**IS NOT**_ overwritten)

## Configuration

The following options may be configured:
- `operations` (default: []): A list of operations to apply to each metric or log resource.
    - `operations[].from` (default: ""): The attribute to copy off of the resource
    - `operations[].to` (default: ""): The destination attribute on each individual metric data point or log

### Example configuration

```yaml
processors:
  resourceattributetransposer:
    operations:
      - from: "some.resource.level.attr"
        to: "some.metricdatapoint.level.attr"
      - from: "another.resource.attr"
        to: "another.datapoint.attr"
```

## Limitations

Currently, this assumes that the resources attributes is a flat map. This means that you cannot move a single resource attribute if it  is under a nested map. You can, however, move a whole nested map.

