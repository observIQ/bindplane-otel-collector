# Webhook Exporter

The webhook exporter sends telemetry data to a webhook endpoint.

## Minimum Agent Versions

<!-- Modify this if we decide to patch release -->

- Introduced: [1.79.0](https://github.com/observIQ/bindplane-otel-collector/releases/tag/v1.79.0)

## Supported Pipelines

- Logs

## How It Works

The webhook exporter sends data to a configured HTTP endpoint. Here's how it processes and sends the data:

1. **Data Collection**: The exporter receives logs from the OpenTelemetry Collector pipeline.

2. **Data Processing**:

   - Data is extracted from the OpenTelemetry data model
   - Each entry's body is parsed as JSON if possible, otherwise kept as a string
   - Batching is handled by the upstream [batch processor](https://github.com/open-telemetry/opentelemetry-collector/blob/main/processor/batchprocessor/README.md).

3. **HTTP Transmission**:

   - Data is sent to the configured endpoint using the specified HTTP method (POST, PATCH, or PUT)
   - The configured Content-Type header is applied
   - Any additional headers specified in the configuration are included
   - The request body contains the data in JSON format

4. **Error Handling**:
   - Failed requests are retried according to the sending queue configuration
   - Non-2xx HTTP responses are treated as errors
   - Connection timeouts are handled according to the configured timeout settings

## Configuration

The following configuration options are available:

| Field            | Type              | Default | Required | Description                                                                                                                                                                                                                                         |
| ---------------- | ----------------- | ------- | -------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| endpoint         | string            |         | `true`   | The URL where the webhook requests will be sent. Must start with http:// or https://                                                                                                                                                                |
| verb             | string            |         | `true`   | The HTTP method to use for the webhook requests. Must be one of: POST, PATCH, PUT                                                                                                                                                                   |
| content_type     | string            |         | `true`   | The Content-Type header for the webhook requests                                                                                                                                                                                                    |
| headers          | map[string]string |         | `false`  | Additional HTTP headers to include in the webhook requests                                                                                                                                                                                          |
| sending_queue    | map               |         | `false`  | Determines how telemetry data is buffered before exporting. See the documentation for the [exporter helper](https://github.com/open-telemetry/opentelemetry-collector/blob/v0.128.0/exporter/exporterhelper/README.md) for more information.        |
| retry_on_failure | map               |         | `false`  | Determines how the exporter will attempt to retry after a failure. See the documentation for the [exporter helper](https://github.com/open-telemetry/opentelemetry-collector/blob/v0.128.0/exporter/exporterhelper/README.md) for more information. |
| timeout          | duration          | 30s     | `false`  | Time to wait per individual attempt to send data to a backend. See the documentation for the [exporter helper](https://github.com/open-telemetry/opentelemetry-collector/blob/v0.128.0/exporter/exporterhelper/README.md) for more information.     |

### Example Configurations

#### Basic Configuration

```yaml
exporters:
  webhook:
    endpoint: https://api.example.com/webhook
    verb: POST
    content_type: application/json
```

#### Sending Queue Configuration

```yaml
exporters:
  webhook:
    endpoint: https://api.example.com/webhook
    verb: POST
    content_type: application/json
    headers:
      X-API-Key: "your-api-key"
    sending_queue:
      enabled: true
      queue_size: 1000
    retry_on_failure:
      enabled: true
      initial_interval: 5s
      max_interval: 30s
      max_elapsed_time: 300s
```

#### Mutual TLS Configuration

```yaml
exporters:
  webhook:
    endpoint: https://api.example.com/webhook
    verb: POST
    content_type: application/json
    headers:
      X-API-Key: "your-api-key"
    tls:
      ca_file: /path/to/ca.pem
      cert_file: /path/to/cert.pem
      key_file: /path/to/key.pem
```

## OCB

This component relies on the `github.com/observiq/bindplane-otel-collector/version` package to get a version value. This version is used to construct a User-Agent header value.

When using this component with the OpenTelemetry Collector Builder (OCB), use the `--ldflags` CLI argument to set the version value at build time. For example:

```sh
builder --config "manifest.yaml" --ldflags "-s -w -X github.com/observiq/bindplane-otel-collector/version.version=v1.2.3"
```

