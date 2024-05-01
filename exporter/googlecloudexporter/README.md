# Google Cloud Exporter

This exporter can be used to send metrics, traces, and logs to Google Cloud Monitoring. It is an extension of the official 
[Google Cloud Exporter](https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/v0.99.0/exporter/googlecloudexporter), with additional processors built in to streamline configuration.

## Configuration
| Field              | Default          | Required | Description                                                                                                                                                                                                  |
|--------------------|------------------|----------|--------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `metric`           |                  | `false`  | The [metric](https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/9fe4230817a192049f6c539f6c5d9ea31ce05f99/exporter/googlecloudexporter#configuration-reference) settings of the exporter. |
| `trace`            |                  | `false`  | The [trace](https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/9fe4230817a192049f6c539f6c5d9ea31ce05f99/exporter/googlecloudexporter#configuration-reference) settings of the exporter.  |
| `log`              |                  | `false`  | The [log](https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/9fe4230817a192049f6c539f6c5d9ea31ce05f99/exporter/googlecloudexporter#configuration-reference) settings of the exporter.    |
| `credentials`      |                  | `false`  | The credentials JSON used to authenticate the GCP client.                                                                                                                                                    |
| `credentials_file` |                  | `false`  | The credentials file used to authenticate the GCP client. Ignored if `credentials` is set.                                                                                                                   |
| `project`          |                  | `false`  | The GCP project used when exporting telemetry data. If not set, the exporter will attempt to extract the value from the specified credentials.                                                               |
| `user_agent`       | `StanzaLogAgent` | `false`  | Overrides the user agent used when making requests.                                                                                                                                                          |
| `timeout`          | `12s`            | `false`  | The timeout for API calls.                                                                                                                                                                                   |
| `sending_queue`    |                  | `false`  | Determines how telemetry data is buffered before exporting.                                                                                                                                                  |
| `batch`            |                  | `false`  | The config of the exporter's [batch processor](https://github.com/open-telemetry/opentelemetry-collector/blob/v0.99.0/processor/batchprocessor).                                                             |
| `append_host`      |                  | `true`   | Append the agent's hostname to incoming telemetry if not already present.                                                                                                                                    |

## Metric Processing Steps
When metric data is received by the Google Cloud Exporter, it is processed in the following steps:

1. **Hostname Detection**: Hostname is appended as an attribute on metrics if not already present.
2. **Batch Processor**: Metrics are batched to decrease the number of requests.
3. **Google Cloud Exporter**: Metrics are exported to GCP.

## Log Processing Steps
When log data is received by the Google Cloud Exporter, it is processed in the following steps:

1. **Hostname Detection**: Hostname is appended as an attribute on logs if not already present.
2. **Batch Processor**: Logs are batched to decrease the number of requests.
3. **Google Cloud Exporter**: Logs are exported to GCP.

## Trace Processing Steps
When trace data is received by the Google Cloud Exporter, it is processed in the following steps:

1. **Hostname Detection**: Hostname is appended as an attribute on traces if not already present.
2. **Batch Processor**: Traces are batched to decrease the number of requests.
3. **Google Cloud Exporter**: Traces are exported to GCP.

## Metric Labels
Unlike the official Google Cloud Exporter, this extension transforms all resource attributes into metric labels by default. Users may still use the `resource_filters` field in the metric config to overwrite this behavior.
