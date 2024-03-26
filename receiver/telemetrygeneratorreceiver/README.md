# Telemetry Generator Receiver
This receiver is used to generate synthetic telemetry for testing and configuration purposes. 

## Minimum Agent Versions
- Introduced: [v1.46.0](https://github.com/observIQ/bindplane-agent/releases/tag/v1.46.0)
- Updated to include host_metrics: [v1.47.0](https://github.com/observIQ/bindplane-agent/releases/tag/v1.47.0)

## Supported Pipelines
- Logs
- Metrics
- Traces

## Configuration for all generators
| Field                | Type      | Default   | Required | Description |
|----------------------|-----------|-----------|----------|-------------|
| payloads_per_second  |  int      |     `1`   | `false`  | The number of payloads this receiver will generate per second.|
| generators           |  list     |           | `true`   | A list of generators to use.|
### Common Generator Configuration
| Field                | Type      | Default          | Required | Description  |
|----------------------|-----------|------------------|----------|--------------|
| type                 |  string   |                  | `true`   | The type of generator to use. Currently `logs`, `otlp`, `metrics`, `host_metrics`, and `windows_events` are supported.  |
| resource_attributes  |  map      |                  | `false`  | A map of resource attributes to be included in the generated telemetry. Values can be `any`.   |
| attributes           |  map      |                  | `false`  | A map of attributes to be included in the generated telemetry. Values can be `any`.  |
| additional_config    |  map      |                  | `false`  | A map of additional configuration options to be included in the generated telemetry. Values can be `any`.|

### Log Generator Configuration
| Field                | Type      | Default | Required | Description |
|----------------------|-----------|---------|----------|-------------|
| body                 |  string   |         | `false`  | The body of the log, set in additional_config |
| severity             |  int      |         | `false`  | The severity of the log message, set in additional_config |

#### Example Configuration
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

### OTLP Replay Generator

The OTLP Replay Generator replays JSON-formatted telemetry from the variable `otlp_json`. It adjusts the timestamps of the telemetry relative the current time, with the most recent record moved to the current time, and the previous records the same relative duration in the past. The `otlp_json` variable should be valid OTLP, such as the JSON created by `plog.JSONMarshaler`,`ptrace.JSONMarshaler`, or `pmetric.JSONMarshaler`. The `otlp_json` variable is set in the `additional_config` section of the generator configuration. The `attributes` and `resource_attributes` fields are ignored.

#### additional_config:

| Field                | Type      | Default          | Required | Description  |
|----------------------|-----------|------------------|----------|--------------|
| telemetry_type       |  string   |                  | `true`   | The type of telemetry to replay: `logs`, `metrics`, or `traces`.  |
| otlp_json            |  string   |                  | `true`  | A string of JSON encoded OTLP telemetry|

#### Example Configuration
```yaml
telemetrygeneratorreceiver:
    payloads_per_second: 1
    generators:
        - type: otlp
          additional_config:
            telemetry_type: "metrics",
			otlp_json:      `{"resourceMetrics":[{"resource":{},"scopeMetrics":[{"scope":{},"metrics":[{"exponentialHistogram":{"dataPoints":[{"attributes":[{"key":"prod-machine","value":{"stringValue":"prod-1"}}],"count":"4","positive":{},"negative":{},"min":0,"max":100}]}}]}]}]}`,
```

### Metrics Generator

The metrics generator creates synthetic metrics. The generator can be configured to create metrics with arbitrary names, values, and attributes. The generator can be configured to create metrics with a random value between a minimum and maximum value, or a constant value by setting `value_max = value_min`. For `Sum` metrics with unit `s` and `Gauge` metrics, the generator will create a random `float` value. For all other `Sum` metrics, the generator will create a random `int` value.

#### additional_config:

| Field                | Type      | Default          | Required | Description  |
|----------------------|-----------|------------------|----------|--------------|
| telemetry_type       |  string   |                  | `true`   | The type of telemetry to replay: `logs`, `metrics`, or `traces`.  |
| metrics           |  array   |                  | `true`  | A list of metrics to generate|

#### metrics:


| Field                | Type      | Default          | Required | Description  |
|----------------------|-----------|------------------|----------|--------------|
| name                 |  string   |                  | `true`   | The metric name  |
| value_min           |  int   |                  | `true`  | The metric's minimum value|
| value_max           |  int   |                  | `true`  | The metric's maximum value|
| type                 |  string   |                  | `true`   | The metric type: `Gauge`, or `Sum`|
| unit                 |  string   |                  | `true`   | The metric unit, either `By`, `by`, `1`, `s`, `{thread}`, `{errors}`, `{packets}`, `{entries}`, `{connections}`, `{faults}`, `{operations}`, or `{processes}`|
| attributes           |  map      |                  | `false`  | A map of attributes to be included in the generated telemetry record. Values can be `any`.|

#### Example Configuration
```yaml
telemetrygeneratorreceiver:
    payloads_per_second: 1
    generators:
        - type: metrics
          resource_attributes:
            host.name: 2ed77de7e4c1
            os.type: linux          
          additional_config:
            metrics: 
            # memory metrics
             - name:	system.memory.usage
                value_min: 100000
                value_max: 1000000000
                type:	Sum
                unit:	By
                attributes:
                  state: cached
            # load metrics                  
              - name:	system.cpu.load_average.1m
                value_min: 0
                value_max: 1
                type: Gauge
                unit:	"{thread}"   
            # file system metrics                                          
              - name: system.filesystem.usage
                value_min: 0
                value_max: 15616700416
                type: Sum
                unit: By
                attributes:
                  device: "/dev/vda1"
                  mode: rw
                  mountpoint: "/etc/hosts"
                  state: reserved
                  type: ext4                    
```


### Host Metrics Generator

The host metrics generator creates synthetic host metrics, from a list of pre-defined metrics. The metrics resource attributes can be set in the `resource_attributes` section of the generator configuration.

#### Example Configuration
```yaml
telemetrygeneratorreceiver:
    payloads_per_second: 1
    generators:
        - type: host_metrics
          resource_attributes:
            host.name: 2ed77de7e4c1
            os.type: linux   
```       
### Windows Events Generator

The Windows Events Generator replays a sample of recorded Windows Event Log data. It has no additional configuration, and will ignore `resource_attributes` and `attributes` fields.

#### Example Configuration
```yaml
telemetrygeneratorreceiver:
    payloads_per_second: 1
    generators:
        - type: windows_events          
```       