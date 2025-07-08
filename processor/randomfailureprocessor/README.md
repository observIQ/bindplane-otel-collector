# Random Failure Processor

The random failure processor is used for testing error resiliency in the collector.

## Supported pipelines

- Logs
- Metrics
- Traces

## How it works

1. The user configures the processor in one or more pipelines.
2. Whenever telemetry passes through the processor, there is a user-configured probability that an error will be returned.

## Configuration

| Field         | Type    | Default          | Required | Description                                                                         |
| ------------- | ------- | ---------------- | -------- | ----------------------------------------------------------------------------------- |
| failure_rate  | float64 | 0.5              | `false`  | The probability, between 0 and 1, of any given piece of telemetry causing an error. |
| error_message | string  | "random failure" | `false`  | The error message that will be returned by the processor.                           |

## Examples

### Usage in pipelines

The random failure processor may be used in a pipeline in order to test what occurs when a section of the pipeline errors:

```yaml
receivers:
  filelog:
    include: [/var/log/logfile.txt]

processors:
  randomfailure:
    failure_rate: 0.1
    error_message: "10% failure occurred!"

exporters:
  nop:

service:
  pipelines:
    logs:
      receivers: [filelog]
      processors: [randomfailure]
      exporters: [nop]
```

In this instance, each log from `/var/log/logfile.txt` has a 10% chance of generating an error when it passes through the processor.
