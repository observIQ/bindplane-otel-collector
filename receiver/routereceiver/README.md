# Route Receiver
This receiver is used to receive telemetry routed from other pipelines.

## Supported pipelines
- Logs
- Metrics
- Traces

## How It Works
1. The user configures this receiver in a pipeline.
2. The user configures a supported component to route telemetry to this receiver.

## Configuration
The route receiver does not have any configuration parameters. It simply receives telemetry from other components.

### Example Config
The following config is an example configuration of the route receiver. In this example, logs are collected from a file and sent to a log count processor. After each minute, the log counts are converted to metrics and sent to the route receiver in the metrics pipeline.
```yaml
receivers:
    filelog:
        include: [./example/apache.log]
    route/log-based-metrics:
processors:
    batch:
    logcount:
        route: log-based-metrics
exporters:
    googlecloud:
    logging:

service:
    pipelines:
        logs:
            receivers: [filelog]
            processors: [batch, logcount]
            exporters: [logging]
        metrics:
            receivers: [route/log-based-metrics]
            processors: [batch]
            exporters: [googlecloud]
```
