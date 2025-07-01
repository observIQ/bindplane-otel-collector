# AWS S3 Event Receiver

The AWS S3 Event Receiver consumes S3 event notifications for object creation events (`s3:ObjectCreated:*`) and emits the S3 object as the string body of a log record.

## How It Works

1. The receiver polls an SQS queue for S3 event notifications.
2. Supports both direct S3 events and S3 events wrapped in SNS notifications (S3 → SNS → SQS).
3. When an object creation event (`s3:ObjectCreated:*`) is received, the receiver downloads the S3 object.
4. The receiver reads the object into the body of a new log record.
5. Non-object creation events are ignored but removed from the queue.
6. If an S3 object is not found (404 error), the corresponding SQS message is preserved for retry later.


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
| notification_type      | string   | s3    | `false`  | The format of notifications in the SQS queue. Valid values: "s3" (direct S3 events), "sns" (S3 events wrapped in SNS notifications) |
| sns_message_format     | object   |       | `false`  | Configuration for parsing SNS messages when notification_type is "sns" |

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
3. Optionally enable "Raw message delivery" on the SNS subscription for simpler parsing.
4. Ensure the collector has permission to read and delete messages from the SQS queue.
5. Ensure the collector has permission to read objects from the S3 bucket.

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

### SNS Message Format Configuration

When `notification_type` is set to "sns", you can configure how to parse SNS messages:

| Field        | Type   | Default | Required | Description |
|--------------|--------|---------|----------|-------------|
| message_field | string | Message | `false`  | The field name in the SNS notification that contains the S3 event |
| format       | string | standard| `false`  | The format of the SNS message. Valid values: "standard" (standard SNS format), "raw" (raw message delivery) |

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
```

### S3 Events via SNS (S3 → SNS → SQS)

```yaml
receivers:
  s3event:
    sqs_queue_url: https://sqs.us-west-2.amazonaws.com/123456789012/my-queue
    notification_type: sns
    sns_message_format:
      message_field: Message  # Default field containing S3 event
      format: standard        # Standard SNS notification format

exporters:
  otlp:
    endpoint: otelcol:4317

service:
  pipelines:
    logs:
      receivers: [s3event]
      exporters: [otlp]
```

### SNS with Raw Message Delivery

```yaml
receivers:
  s3event:
    sqs_queue_url: https://sqs.us-west-2.amazonaws.com/123456789012/my-queue
    notification_type: sns
    sns_message_format:
      format: raw  # Raw message delivery enabled on SNS subscription

exporters:
  otlp:
    endpoint: otelcol:4317

service:
  pipelines:
    logs:
      receivers: [s3event]
      exporters: [otlp]
```

### SNS with Custom Message Field

```yaml
receivers:
  s3event:
    sqs_queue_url: https://sqs.us-west-2.amazonaws.com/123456789012/my-queue
    notification_type: sns
    sns_message_format:
      message_field: CustomS3Event  # Custom field name containing S3 event
      format: standard

exporters:
  otlp:
    endpoint: otelcol:4317

service:
  pipelines:
    logs:
      receivers: [s3event]
      exporters: [otlp]
```
