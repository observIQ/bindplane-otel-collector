# Webhook Exporter
<!-- TODO: write docs -->

## Minimum Agent Versions

<!-- TODO: update once released -->
- Introduced: [vx.xx.x](docslink)

## Supported Pipelines

- Logs
<!-- TODO: update once more pipelines are supported -->

## How It Works

## Configuration

| Field        | Type              | Default | Required | Description                                                                          |
|--------------|-------------------|---------|----------|--------------------------------------------------------------------------------------|
| endpoint     | string            |         | `true`   | The URL where the webhook requests will be sent. Must start with http:// or https:// |
| verb         | string            |         | `true`   | The HTTP method to use for the webhook requests. Must be one of: POST, PATCH, PUT    |
| content_type | string            |         | `true`   | The Content-Type header for the webhook requests                                     |
| headers      | map[string]string |         | `false`  | Additional HTTP headers to include in the webhook requests                           |

