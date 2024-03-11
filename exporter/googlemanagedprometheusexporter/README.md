# Google Managed Prometheus Exporter

This exporter can be used to send metrics to Google Cloud Managed Service for Prometheus. It is an extension of the official 
[Google Managed Prometheus Exporter](https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/v0.96.0/exporter/googlemanagedprometheusexporter), with additional configuration options.

## Supported pipelines
- Metrics

## How It Works
1. The user configures this exporter in a pipeline
2. If the pipeline does not use the prometheus receiver with the gcp detector, set resource attributes in the pipeline [as described in the upstream documentation](https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/v0.96.0/exporter/googlemanagedprometheusexporter#resource-attribute-handling).
3. Metrics are sent to Google Cloud.

## Configuration
| Field              | Type    | Default          | Required | Description                                                                                                                                                                                                                                 |
|--------------------|---------|------------------|----------|---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `credentials_file` | string  |                  | `false`  | The [credentials file](https://developers.google.com/workspace/guides/create-credentials#service-account) used to authenticate the GPM client. Ignored if `credentials` is set.                                                             |
| `credentials`      | string  |                  | `false`  | The [credentials JSON](https://developers.google.com/workspace/guides/create-credentials#service-account) used to authenticate the GPM client.                                                                                              |
| `metric.endpoint`  | string  |                  | `false`  | The endpoint where metric data is sent to.                                                                                                                                                                                                  |
| `metric`           | object  |                  | `false`  | The metric settings of the exporter.                                                                                                                                                                                                        |
| `project`          | string  |                  | `false`  | The GCP project used when exporting telemetry data. If not set, the exporter will attempt to extract the value from the specified credentials.                                                                                              |
| `use_insecure`     | boolean | `false`          | `false`  | Uses gRPC communication if true. Only has an effect if the endpoint is set.                                                                                                                                                                 |
| `user_agent`       | string  | `StanzaLogAgent` | `false`  | Overrides the user agent used when making requests.                                                                                                                                                                                         |
| `sending_queue`    | object  |                  | `false`  | Determines how telemetry data is buffered before exporting. See the documentation for the [exporter helper](https://github.com/open-telemetry/opentelemetry-collector/blob/v0.96.0/exporter/exporterhelper/README.md) for more information. |

## Example configuration
This configuration scrapes the agent's self metrics, using a credentials file to authenticate.
```yaml
receivers:
  # Scrape the agent's self metrics with the prometheus receiver
  prometheus:
    config:
      scrape_configs:
        - job_name: 'otel-collector'
          scrape_interval: 30s
          static_configs:
            - targets: ['0.0.0.0:8888']
processors:
  batch:
  # The location label is required. Here we are specifying use us-east1-a.
  # "namespace" and "cluster" may also be set here, if desired.
  # Alternatively, the resourcedetection processor may be used with the "gcp" detector if running in gcp.
  resource:
    attributes:
      # Add a location 
      - key: "location"
        value: "us-east1-a"
        action: "upsert"

exporters:
  googlemanagedprometheus:
    credentials_file: ${CREDENTIALS_FILE_PATH}

service:
  pipelines:
    metrics:
      receivers: [prometheus]
      processors: [resource, batch]
      exporters: [googlemanagedprometheus]

```
