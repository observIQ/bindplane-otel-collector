# Bindplane Extension

This extension is used by Bindplane in custom distributions to store Bindplane specific information.

- It is used for custom collector distributions and BDOT v2, but crucially not BDOT v1.

## Configuration

| Field                         | Type   | Default | Required | Description                                                                                                      |
| ----------------------------- | ------ | ------- | -------- | ---------------------------------------------------------------------------------------------------------------- |
| opamp                         | string |         | `false`  | Component ID of an OpAMP extension. Needed to generate custom messages for throughput and topology measurements. |
| measurements_interval         | time   |         | `false`  | Interval on which to report measurements. Reporting is disabled if the duration is 0.                            |
| topology_interval             | time   |         | `false`  | Interval on which to report topology. Reporting is disabled if the duration is 0.                                |
| extra_measurements_attributes | map    |         | `false`  | Key-value pairs to add to all reported measurements.                                                             |

## Example Configuration

Bindplane expects a single unnamed bindplane extension in the configuration.

```yaml
receivers:
  nop:

exporters:
  nop:

extensions:
  bindplane:
    opamp: opamp
    measurements_interval: 1m
    topology_interval: 1m
    extra_measurements_attributes:
      - "keyA": "valueB"
  opamp:
    server:
      ws:
        endpoint: ws://127.0.0.1:64333/v1/opamp

service:
  extensions: [bindplane, opamp]
  pipelines:
    logs:
      receivers: [nop]
      exporters: [nop]
```

In this configuration we specify an OpAMP extension, measurement & topology intervals of 1 minute, and one set of extra measurement attributes.
