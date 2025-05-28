# AWS S3 Event Receiver

The AWS S3 Event Receiver consumes S3 event notifications for object creation events (`s3:ObjectCreated:*`) and emits the S3 object as the string body of a log record.

## How It Works

1. The receiver polls an SQS queue for S3 event notifications.
2. When an object creation event (`s3:ObjectCreated:*`) is received, the receiver downloads the S3 object.
3. The receiver reads the object into the body of a new log record.
4. Non-object creation events are ignored but removed from the queue.
5. If an S3 object is not found (404 error), the corresponding SQS message is preserved for retry later.


## Configuration

| Field                  | Type   | Default | Required | Description |
|------------------------|--------|---------|----------|-------------|
| sqs_queue_url          | string |         | `true`   | The URL of the SQS queue to poll for S3 event notifications (the AWS region is automatically extracted from this URL) |
| standard_poll_interval | duration | 15s   | `false`  | The interval at which the SQS queue is polled for messages |
| max_poll_interval      | duration | 120s   | `false`  | The maximum interval at which the SQS queue is polled for messages |
| polling_backoff_factor | float    | 2     | `false`  | The factor by which the polling interval is multiplied after an unsuccessful poll |
| workers                | int      | 5     | `false`  | The number of workers to process messages in parallel |
| visibility_timeout     | duration | 300s  | `false`  | The visibility timeout for SQS messages |
| max_log_size           | int      | 1048576  | `false`  | The maximum size of a log record in bytes. Logs exceeding this size will be split |
| max_logs_emitted       | int      | 1000  | `false`  | The maximum number of log records to emit in a single batch. A higher number will result in fewer batches, but more memory |

## AWS Setup

To use this receiver, you need to:

1. Ensure the collector has permission to read and delete messages from the SQS queue.
2. Ensure the collector has permission to read objects from the S3 bucket.

## Example Configuration

```yaml
receivers:
  s3event:
    sqs_queue_url: https://sqs.us-west-2.amazonaws.com/123456789012/my-queue

exporters:
  otlp:
    endpoint: otelcol:4317

service:
  pipelines:
    logs:
      receivers: [s3event]
      exporters: [otlp]
```
