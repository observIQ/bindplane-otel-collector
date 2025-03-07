# AWS S3 Event Receiver

The AWS S3 Event Receiver consumes S3 event notifications for object creation events and emits the S3 object as the string body of a log record.

## How It Works

1. The receiver polls an SQS queue for S3 event notifications.
2. When an object creation event is received, the receiver downloads the S3 object.
3. The receiver reads the object into the body of a new log record.

## TODO

- Instead of reading the entire object into a log body, stream the object content and tokenize it into multiple logs.


## Configuration

| Field                  | Type   | Default | Required | Description |
|------------------------|--------|---------|----------|-------------|
| sqs_queue_url          | string |         | `true`   | The URL of the SQS queue to poll for S3 event notifications (the AWS region is automatically extracted from this URL) |
| standard_poll_interval | duration | 15s   | `false`  | The interval at which the SQS queue is polled for messages |
| max_poll_interval      | duration | 2m   | `false`  | The maximum interval at which the SQS queue is polled for messages |
| polling_backoff_factor | float    | 2     | `false`  | The factor by which the polling interval is multiplied after an unsuccessful poll |
| workers                | int      | 5     | `false`  | The number of workers to process messages in parallel |
| visibility_timeout     | duration | 300s  | `false`  | The visibility timeout for SQS messages |
| api_max_messages       | int    | 10      | `false`  | The maximum number of messages to retrieve in a single SQS poll |
| delete_after_processing | bool   | true    | `false`  | Whether to delete SQS messages after processing |
| preserve_s3_objects    | bool   | false   | `false`  | Whether to preserve S3 objects after processing |

## AWS Setup

To use this receiver, you need to:

1. Configure an S3 bucket to send event notifications to an SQS queue for object creation events.
2. Ensure the collector has permission to read and delete messages from the SQS queue.
3. Ensure the collector has permission to read objects from the S3 bucket.

## Example Configuration

```yaml
receivers:
  awss3event:
    sqs_queue_url: https://sqs.us-west-2.amazonaws.com/123456789012/my-queue

exporters:
  otlp:
    endpoint: otelcol:4317

service:
  pipelines:
    logs:
      receivers: [awss3event]
      exporters: [otlp]
```
