# Telemetry Generator Receiver
This receiver is used to generate synthetic telemetry for testing and configuration purposes. 

## Minimum Agent Versions
- Introduced: [v1.46.0](https://github.com/observIQ/bindplane-agent/releases/tag/v1.46.0)

## Supported Pipelines
- Logs
- Metrics
- Traces

## How It Works
1. The user configures this receiver in a pipeline.
2. The user configures a supported component to route telemetry from this receiver.

## Configuration
| Field                | Type      | Default   | Required | Description |
|----------------------|-----------|-----------|----------|-------------|
| payloads_per_second  |  int      |     `1`   | `false`  | The number of payloads this receiver will generate per second.|
| generators           |  list     |           | `true`   | A list of generators to use.|
### Generator Configuration
| Field                | Type      | Default          | Required | Description  |
|----------------------|-----------|------------------|----------|--------------|
| type                 |  string   |                  | `true`   | The type of generator to use. Currently only `logs` is supported.  |
| resource_attributes  |  map      |                  | `false`  | A map of resource attributes to be included in the generated telemetry. Values can be `any`.   |
| attributes           |  map      |                  | `false`  | A map of attributes to be included in the generated telemetry. Values can be `any`.  |
| additional_config    |  map      |                  | `false`  | A map of additional configuration options to be included in the generated telemetry. Values can be `any`.|

### Log Generator Configuration
| Field                | Type      | Default | Required | Description |
|----------------------|-----------|---------|----------|-------------|
| body                 |  string   |         | `false`  | The body of the log, set in additional_config |
| severity             |  int      |         | `false`  | The severity of the log message, set in additional_config |

### Example Configuration
```yaml
telemetrygeneratorreceiver:
    payloads_per_second: 1
    generators:
        - type: logs
          resource_attributes:
              res_key1: res_value1
              res_key2: res_value2            
          attributes:
              attr_key1: attr_value1
              attr_key2: attr_value2            
          additional_config:
              body: this is the body   
              severity: 4
        - type: logs
          resource_attributes:
              res_key3: res_value3
              res_key4: res_value4            
          attributes:
              attr_key3: attr_value3
              attr_key4: attr_value4            
          additional_config:
              body: this is another body   
              severity: 10
```
