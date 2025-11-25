# HTTP Receiver
This receiver is capable of collecting logs for a variety of services, serving as a default HTTP log receiver. Anything that is able to send logs to an endpoint using HTTP will be able to utilize this receiver.

## Minimum Agent Versions
- Introduced: [v1.39.0](https://github.com/observIQ/bindplane-otel-collector/releases/tag/v1.39.0)

## Supported Pipelines
- Logs

## How It Works
1. The user configures this receiver in a pipeline.
2. The user configures a supported component to route telemetry from this receiver.

## Prerequisites
- The log source can be configured to send logs to an endpoint using HTTP

## Supported Formats

The receiver automatically detects and handles different payload formats based on the `Content-Type` header:

### JSON Payloads (`Content-Type: application/json`)
- **JSON Object**: Single log entry as a JSON object
  ```json
  {"message": "error occurred", "level": "error", "timestamp": 1699276151086}
  ```
- **JSON Array**: Multiple log entries as an array of JSON objects
  ```json
  [
    {"message": "first log", "level": "info"},
    {"message": "second log", "level": "debug"}
  ]
  ```

### Text Payloads (any other `Content-Type` or no header)
Plain text payloads are automatically wrapped in a log structure with a `body` field:
```
This is a plain text log message
```

**Note**: If `Content-Type: application/json` is specified but the payload is not valid JSON, the request will be rejected with a 422 status code.

## Configuration
| Field                | Type      | Default          | Required | Description                                                                                                                                                                            |
|----------------------|-----------|------------------|----------|----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| endpoint             |  string   |                  | `true`   | The hostname and port the receiver should listen on for logs being sent as HTTP POST requests.                                                                                         |
| path                 |  string   |                  | `false`  | Specifies a path the receiver should be listening to for logs. Useful when the log source also sends other data to the endpoint, such as metrics.                                      |
| tls.key_file         |  string   |                  | `false`  | Configure the receiver to use TLS.                                                                                                                                                     |
| tls.cert_file        |  string   |                  | `false`  | Configure the receiver to use TLS.                                                                                                                                                     |

### Example Configuration
```yaml
receivers:
  http:
    endpoint: "localhost:12345"
    path: "/api/v2/logs"
exporters:
  googlecloud:
    project: my-gcp-project

service:
  pipelines:
    logs:
      receivers: [http]
      exporters: [googlecloud]
```

### Example Configuration With TLS
```yaml
receivers:
  http:
    endpoint: "0.0.0.0:12345"
    path: "/logs"
    tls:
      key_file: "certs/server.key"
      cert_file: "certs/server.crt"
exporters:
  googlecloud:
    project: my-gcp-project

service:
  pipelines:
    logs:
      receivers: [http]
      exporters: [googlecloud]
```
