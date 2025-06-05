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
| retry_on_failure | map    |         | Configuration for retry behavior                                            |
| enabled         | bool    | true    | Whether to enable retry on failure                                          |
| initial_interval| string  | 5s      | Time to wait after first failure before retrying                            |
| max_interval    | string  | 30s     | Upper bound on backoff                                                      |
| max_elapsed_time| string  | 300s    | Maximum time spent trying to send a batch (0 = no limit)                    |
| multiplier      | float64 | 1.5     | Factor by which retry interval is multiplied on each attempt                |
| sending_queue   | map     |         | Configuration for the sending queue                                         |
| enabled         | bool    | true    | Whether to enable the sending queue                                         |
| num_consumers   | int     | 10      | Number of consumers that dequeue batches                                    |
| wait_for_result | bool    | false   | Whether to block until request is processed                                 |
| block_on_overflow| bool   | false   | Whether to block when queue is full                                         |
| sizer           | string  | requests| How queue and batching is measured (requests/items/bytes)                   |
| queue_size      | int     | 1000    | Maximum size the queue can accept                                           |
| batch           | map     |         | Configuration for batching behavior                                         |
| flush_timeout   | string  |         | Time after which a batch will be sent regardless of size                    |
| min_size        | int     |         | Minimum size of a batch                                                     |
| max_size        | int     |         | Maximum size of a batch (enables batch splitting)                           |
| timeout         | string  | 5s      | Time to wait per individual attempt to send data                            |

Note: Time values (initial_interval, max_interval, max_elapsed_time, timeout) accept duration strings with units: "ns", "us" (or "µs"), "ms", "s", "m", "h".

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

