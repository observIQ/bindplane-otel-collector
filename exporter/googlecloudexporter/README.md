# Google Cloud Exporter

This exporter can be used to send metrics, traces, and logs to Google Cloud Monitoring. It is an extension of the official 
[Google Cloud Exporter](https://github.com/observIQ/opentelemetry-collector-contrib/tree/5be4317b53b925df35b7845cac3cb174c2e007a0/exporter/googlecloudexporter), with additional processors built in to streamline configuration.

## Configuration
| Field               | Default               | Required | Description |
| ---                 | ---                   | ---      | ---         |
| `credentials`       |                       | `false`  | The credentials JSON used to authenticate the GCP client. |
| `credentials_file`  |                       | `false`  | The credentials file used to authenticate the GCP client. Ignored if `credentials` is set. |
| `project`           |                       | `false`  | The GCP project used when exporting telemetry data. |
| `endpoint`          |                       | `false`  | The endpoint used when exporting telemetry data. |
| `location`          | `global`              | `false`  | The GCP location attribute attached to telemetry data. |
| `namespace`         | `{hostname}`          | `false`  | The GCP namespace attribute attached to telemetry data. |
| `user_agent`        | `observIQ-otel-agent` | `false`  | Overrides the user agent used when making requests. |
| `timeout`           | `12s`                 | `false`  | The timeout for API calls. |
| `resource_mappings` | [See below](#resource-mapping-default)         | `false`  | Defines a mapping of resources from source to target. |
| `retry_on_failure`  |                       | `false`  | Handle retries when sending data to Google Cloud fails. |
| `sending_queue`     |                       | `false`  | Determines how telemetry data is buffered before exporting. |
| `batch`             |                       | `false`  | The config of the exporter's [batch processor](https://github.com/open-telemetry/opentelemetry-collector/tree/v0.53.0/processor/batchprocessor). |
| `normalize`         |                       | `false`  | The config of the exporter's [normalize sums processor](https://github.com/GoogleCloudPlatform/opentelemetry-operations-collector/tree/master/processor/normalizesumsprocessor). |
| `detector`          |                       | `false`  | The config of the exporter's [reseource detection processor](https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/v0.53.0/processor/resourcedetectionprocessor). |
| `attributer`        |                       | `false`  | The config of the exporter's [resource processor](https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/v0.53.0/processor/resourceprocessor). |
| `transposer`        |                       | `false`  | The config of the exporter's [resource transposer processor](https://github.com/observIQ/observiq-otel-collector/tree/main/processor/resourceattributetransposerprocessor). |

### Resource Mapping Default
```yaml
- target_type: generic_node
  label_mappings:
    - source_key: host.name
      target_key: node_id
    - source_key: location
      target_key: location
    - source_key: namespace
      target_key: namespace
```

## Metric Processing Steps
When metric data is received by the Google Cloud Exporter, it is processed in the following steps:

1. **Resource Detection Processor**: Hostname is appended as an attribute on metrics.
2. **Resource Processor**: Location and namespace are appened as  attributes on metrics.
3. **Transposer Processor**: Resource attributes are moved to metric attributes.
4. **Normalize Sums Processor**: Counter based metrics are normalized to avoid abnormal rendering in GCP.
5. **Batch Processor**: Metrics are batched to decrease the number of requests.
6. **Google Cloud Exporter**: Metrics are exported to GCP.

## Log Processing Steps
When log data is received by the Google Cloud Exporter, it is processed in the following steps:

1. **Resource Detection Processor**: Hostname is appended as an attribute on logs.
2. **Resource Processor**: Location and namespace are appened as  attributes on logs.
3. **Batch Processor**: Logs are batched to decrease the number of requests.
4. **Google Cloud Exporter**: Logs are exported to GCP.

## Trace Processing Steps
When trace data is received by the Google Cloud Exporter, it is processed in the following steps:

1. **Resource Detection Processor**: Hostname is appended as an attribute on traces.
2. **Resource Processor**: Location and namespace are appened as  attributes on traces.
3. **Batch Processor**: Traces are batched to decrease the number of requests.
4. **Google Cloud Exporter**: Traces are exported to GCP.
