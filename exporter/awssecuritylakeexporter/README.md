# AWS Security Lake Exporter

This exporter sends OCSF-formatted logs as Parquet files to an AWS Security Lake S3 bucket. It is designed to integrate with OpenTelemetry collectors to export log telemetry data into Amazon Security Lake for centralized security data management.

## Supported Pipelines

- Logs

## How It Works

1. The exporter receives OCSF-formatted log data from the collector pipeline.
2. It converts the logs into Parquet format.
3. It writes the Parquet files to the configured AWS Security Lake S3 bucket using the appropriate partition path.

## Prerequisites

- Logs must be in OCSF format before reaching this exporter. Use the [OCSF processor](../../processor/ocsfprocessor/README.md) upstream in the pipeline to transform logs into OCSF format.
- An AWS Security Lake custom source must be registered in your AWS account.
- AWS credentials must be available via the standard AWS credential chain (environment variables, shared credentials file, IAM role, etc.).

## Configuration

| Field            | Type   | Default | Required | Description                                                          |
| ---------------- | ------ | ------- | -------- | -------------------------------------------------------------------- |
| `region`         | string |         | `true`   | The AWS region where the Security Lake S3 bucket resides.            |
| `s3_bucket`      | string |         | `true`   | The name of the Security Lake S3 bucket.                             |
| `s3_prefix`      | string | `ext/`  | `false`  | The S3 key prefix for uploaded objects.                              |
| `source_name`    | string |         | `true`   | The custom source name registered in Security Lake.                  |
| `account_id`     | string |         | `true`   | The AWS account ID used in the partition path.                       |
| `role_arn`       | string |         | `false`  | An optional IAM role ARN to assume for S3 writes.                    |
| `endpoint`       | string |         | `false`  | An optional custom endpoint for S3 writes (useful for testing).      |
| `retry_on_failure` | object | | `false` | Standard OpenTelemetry retry configuration. |
| `sending_queue`  | object |         | `false`  | Standard OpenTelemetry queue/batch configuration.                    |
| `timeout`        | duration | `5s`  | `false`  | The timeout for S3 write operations.                                 |

## Example Configuration

### Basic Configuration

```yaml
aws_security_lake:
  region: "us-east-1"
  s3_bucket: "aws-security-data-lake-us-east-1-xxxxxxxxxxxx"
  source_name: "my-custom-source"
  account_id: "123456789012"
```

### Configuration with IAM Role

```yaml
aws_security_lake:
  region: "us-east-1"
  s3_bucket: "aws-security-data-lake-us-east-1-xxxxxxxxxxxx"
  source_name: "my-custom-source"
  account_id: "123456789012"
  role_arn: "arn:aws:iam::123456789012:role/SecurityLakeWriteRole"
```

### Full Pipeline Example

```yaml
receivers:
  filelog:
    include:
      - /var/log/*.log

processors:
  ocsf:
    # Transform logs into OCSF format

exporters:
  aws_security_lake:
    region: "us-east-1"
    s3_bucket: "aws-security-data-lake-us-east-1-xxxxxxxxxxxx"
    source_name: "my-custom-source"
    account_id: "123456789012"

service:
  pipelines:
    logs:
      receivers: [filelog]
      processors: [ocsf]
      exporters: [aws_security_lake]
```
