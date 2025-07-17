# AWS S3 Event Receiver

The AWS S3 Event Receiver consumes S3 event notifications for object creation events (`s3:ObjectCreated:*`) and emits the S3 object as the string body of a log record.

## How It Works

1. The receiver polls an SQS queue for S3 event notifications.
2. Supports both direct S3 events and S3 events wrapped in SNS notifications (S3 → SNS → SQS).
3. When an object creation event (`s3:ObjectCreated:*`) is received, the receiver downloads the S3 object.
4. The receiver reads the object into the body of a new log record.
5. Non-object creation events are ignored but removed from the queue.
6. If an S3 object is not found (404 error), the corresponding SQS message is preserved for retry later.

## Visibility Extension Behavior

The receiver implements a sophisticated visibility extension strategy to handle long-running processing:

1. **Initial Visibility**: When a message is received, it becomes invisible for the duration specified by `visibility_timeout` (default: 5 minutes).

2. **Regular Extensions**: The receiver extends the visibility window by `visibility_extension_interval` (default: 1 minute) before the current window expires.

3. **Maximum Window**: Extensions stop when the total visibility time reaches `max_visibility_window` (default: 1 hour). SQS has a max window of 12 hours, and this allows the receiver to set a shorter maximum window.

4. **Safety Margins**: The receiver always extends calls to extend the visibility window 80% of the way through the current window.  This helps prevent race conditions where the message may become visible before the window has been extended.

This approach ensures that:

- Messages remain invisible during processing
- Long-running operations don't cause message expiration
- Messages eventually become visible if processing takes too long
- The system respects SQS's 12-hour visibility limit

## Configuration

| Field                            | Type     | Default    | Required | Description |
|----------------------------------|----------|------------|----------|-------------|
| sqs_queue_url                    | string   |            | `true`   | The URL of the SQS queue to poll for S3 event notifications (the AWS region is automatically extracted from this URL) |
| standard_poll_interval           | duration | 15s        | `false`  | The interval at which the SQS queue is polled for messages |
| max_poll_interval                | duration | 120s       | `false`  | The maximum interval at which the SQS queue is polled for messages |
| polling_backoff_factor           | float    | 2          | `false`  | The factor by which the polling interval is multiplied after an unsuccessful poll |
| workers                          | int      | 5          | `false`  | The number of workers to process messages in parallel |
| visibility_timeout               | duration | 5m         | `false`  | The visibility timeout for SQS messages |
| visibility_extension_interval    | duration | 1m         | `false`  | How often to extend message visibility during processing. Should be less than visibility_timeout.  Minimum is 10s. |
| max_visibility_window            | duration | 1h         | `false`  | Maximum total time a message can remain invisible before becoming visible to other consumers. Must be less than SQS's 12-hour limit |
| max_log_size                     | int      | 1048576    | `false`  | The maximum size of a log record in bytes. Logs exceeding this size will be split |
| max_logs_emitted                 | int      | 1000       | `false`  | The maximum number of log records to emit in a single batch. A higher number will result in fewer batches, but more memory |
| notification_type                | enum     | s3         | `false`  | The Notification Type that the receiver expects.  Valid values are `s3` or `sns` |

## AWS Setup

### Direct S3 Events Setup

To use this receiver with direct S3 events (S3 → SQS), you need to:

1. Configure S3 bucket event notifications to send directly to an SQS queue.
2. Ensure the collector has permission to read and delete messages from the SQS queue.
3. Ensure the collector has permission to read objects from the S3 bucket.

### SNS Integration Setup (S3 → SNS → SQS)

To use this receiver with SNS integration, you need to:

1. Configure S3 bucket event notifications to send to an SNS topic.
2. Subscribe an SQS queue to the SNS topic.
3. Ensure the collector has permission to read and delete messages from the SQS queue.
4. Ensure the collector has permission to read objects from the S3 bucket.

### Required IAM Permissions

```json
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Effect": "Allow",
            "Action": [
                "sqs:ReceiveMessage",
                "sqs:DeleteMessage"
            ],
            "Resource": "arn:aws:sqs:REGION:ACCOUNT:QUEUE-NAME"
        },
        {
            "Effect": "Allow",
            "Action": [
                "s3:GetObject"
            ],
            "Resource": "arn:aws:s3:::BUCKET-NAME/*"
        }
    ]
}
```

## Example Configurations

### Direct S3 Events (Default)

```yaml
receivers:
  s3event:
    sqs_queue_url: https://sqs.us-west-2.amazonaws.com/123456789012/my-queue
    notification_type: s3  # Default, can be omitted

exporters:
  otlp:
    endpoint: otelcol:4317

service:
  pipelines:
    logs:
      receivers: [s3event]
      exporters: [otlp]
```### S3 Events via SNS (S3 → SNS → SQS)

```yaml
receivers:
  s3event:
    sqs_queue_url: https://sqs.us-west-2.amazonaws.com/123456789012/my-queue
    notification_type: sns

exporters:
  otlp:
    endpoint: otelcol:4317

service:
  pipelines:
    logs:
      receivers: [s3event]
      exporters: [otlp]
```
