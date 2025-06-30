# AWS S3 Event Extension

The AWS S3 Event Extension downloads newly created S3 objects to a specified directory. It reads from an SQS queue, consuming events which indicate that a new object has been created in S3. It supports multiple SQS notification formats, including `s3:ObjectCreated:*` (default) and Crowdstrike FDR.

## Configuration

| Field                  | Type   | Default | Required | Description |
|------------------------|--------|---------|----------|-------------|
| sqs_queue_url          | string |         | `true`   | The URL of the SQS queue to poll for S3 event notifications (the AWS region is automatically extracted from this URL) |
| standard_poll_interval | duration | 15s   | `false`  | The interval at which the SQS queue is polled for messages |
| max_poll_interval      | duration | 120s   | `false`  | The maximum interval at which the SQS queue is polled for messages |
| polling_backoff_factor | float    | 2     | `false`  | The factor by which the polling interval is multiplied after an unsuccessful poll |
| workers                | int      | 5     | `false`  | The number of workers to process messages in parallel |
| visibility_timeout     | duration | 300s  | `false`  | The visibility timeout for SQS messages |
| event_format           | string   | "aws_s3"    | `false`   | The format of the S3 event notifications. Valid values are `aws_s3` (default) and `crowdstrike_fdr`. |
| directory              | string   | ""    | `true`   | The directory to which objects will be downloaded. |

### How `directory` is used

The collector must have write access to the directory specified in `directory`. Within this directory, the extension will create new subdirectories to mirror the buckets where new objects are created.

### Disambiguation of temporarily download files

The extension will download files to a temporary file name which ends with `.bptmp`. Once the file has been downloaded and the extension has verified that it is a valid object, the temporary file will be renamed to the actual file name.

If using the `filelog` receiver to read the files, it is recommended that you `exclude` files ending with `.bptmp`. e.g. If using `directory: /tmp/s3event`, you should use `include: /tmp/s3event/**/*` and `exclude: /tmp/s3event/**/*.bptmp`. It is also recommended to set `delete_after_read: true` in the `filelog` receiver so downloaded files are cleaned up after they are consumed.

## AWS Setup

### AWS S3 Event Notifications

To use this extension with `event_format: aws_s3`, you need to:

1. Configure an S3 bucket to send event notifications to an SQS queue for object creation events.
   - Configure your S3 event notifications with `BatchSize: 1` to ensure each SQS message contains only one S3 event.
   - This setting is crucial because if an object cannot be accessed (e.g., 404 error), the entire SQS message is preserved for retry.
   - If a message contains multiple objects and one fails, all objects will be reprocessed on retry, causing unnecessary duplication.
2. Ensure the collector has permission to read and delete messages from the SQS queue.
3. Ensure the collector has permission to read objects from the S3 bucket.

### Crowdstrike FDR

To use this extension with `event_format: crowdstrike_fdr`, you need to:

1. Configure a Crowdstrike FDR to send events to an SQS queue.
2. Ensure the collector has permission to read and delete messages from the SQS queue.
3. Ensure the collector has permission to read objects from the S3 bucket.

## Example Configuration

```yaml
extensions:
  s3event:
    sqs_queue_url: https://sqs.us-west-2.amazonaws.com/123456789012/my-queue
    directory: /tmp/s3event

receivers:
  filelog:
    include: /tmp/s3event/**/*
    exclude: /tmp/s3event/**/*.bptmp
    delete_after_read: true

exporters:
  otlp:
    endpoint: otelcol:4317

service:
  extensions: [s3event]
  pipelines:
    logs:
      receivers: [filelog]
      exporters: [otlp]
```
