# Webhook Exporter

The webhook exporter sends telemetry data to a webhook endpoint.

## Minimum Agent Versions

<!-- TODO: update once released -->
- Introduced: [vx.xx.x](docslink)

## Supported Pipelines

- Logs
<!-- TODO: update once more pipelines are supported -->

## How It Works

## Configuration

The following configuration options are available:

| Field         | Type              | Default | Required | Description                                                                          |
|---------------|-------------------|---------|----------|--------------------------------------------------------------------------------------|
| endpoint      | string            |         | `true`   | The URL where the webhook requests will be sent. Must start with http:// or https:// |
| verb          | string            |         | `true`   | The HTTP method to use for the webhook requests. Must be one of: POST, PATCH, PUT    |
| content_type  | string            |         | `true`   | The Content-Type header for the webhook requests                                     |
| headers       | map[string]string |         | `false`  | Additional HTTP headers to include in the webhook requests                           |
| sending_queue | map               |         | `false`  | Determines how telemetry data is buffered before exporting. See the documentation for the [exporter helper](https://github.com/open-telemetry/opentelemetry-collector/blob/v0.127.0/exporter/exporterhelper/README.md) for more information.               |

### Queue Settings

The `sending_queue` configuration allows you to control how telemetry data is buffered before being exported. The following options are available:

| Field           | Type    | Default | Description                                                                 |
|-----------------|---------|---------|-----------------------------------------------------------------------------|
| enabled         | bool    | true    | Whether to enable the sending queue                                         |
| queue_size      | int     | 5000    | Maximum number of batches kept in memory before dropping                    |
| num_consumers   | int     | 10      | Number of consumers that dequeue batches                                    |
| storage         | string  | memory  | Storage type for the queue (memory or persistent)                           |
| persistent      | map     |         | Configuration for persistent storage (only used if storage is "persistent") |

Example configuration:

```yaml
exporters:
  webhook:
    logs:
      endpoint: "http://localhost:8081"
      verb: "POST"
      content_type: "application/json"
      sending_queue:
        enabled: true
        queue_size: 1000
        num_consumers: 5
```

