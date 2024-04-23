# AWS S3 Rehydration Receiver
Rehydrates OTLP from AWS S3 that was stored using the [awss3exporter](https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/v0.98.0/exporter/awss3exporter/README.md).

## Important Note
This is not a traditional receiver that continually produces data but rather rehydrates all objects found within a specified time range. Once all of the objects have been rehydrated in that time range the receiver will stop producing data. After the receiver has detected three consecutive empty polls it will stop polling for new objects in order to prevent unnecessary API calls.

## Minimum Agent Versions
- Introduced: [v1.49.0](https://github.com/observIQ/bindplane-agent/releases/tag/v1.49.0)

## Supported Pipelines
- Metrics
- Logs
- Traces

## How it works
1. The receiver polls S3 for all objects in the specified bucket.
2. The receiver will parse each object's path to determine if it matches a path created by the [AWS S3 Exporter](https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/v0.98.0/exporter/awss3exporter/README.md#example-configuration).
3. If the object path is from the exporter, the receiver will parse the timestamp represented by the path.
4. If the timestamp is within the configured range the receiver will download the object and parse its contents into OTLP data.

    a. The receiver will process both uncompressed JSON objects and objects compressed with gzip.

## Configuration
| Field              | Type      | Default          | Required | Description                                                                                                                                                                            |
|--------------------|-----------|------------------|----------|----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| region             |  string   |                  | `true`   | The AWS Region of the bucket to rehydrate from.                                                         |
| s3_bucket          |  string   |                  | `true`   | The name of the bucket to rehydrate from.                                                                                                                                           |
| s3_prefix          |  string   |                  | `false`  | The prefix for the S3 key (root directory inside bucket). Should match the `s3_prefix` value of the AWS S3 Exporter.                                                                          |
| starting_time      |  string   |                  | `true `  | The UTC start time that represents the start of the time range to rehydrate from. Must be in the form `YYYY-MM-DDTHH:MM`.                                                              |
| ending_time        |  string   |                  | `true `  | The UTC end time that represents the end of the time range to rehydrate from. Must be in the form `YYYY-MM-DDTHH:MM`.                                                                  |
| delete_on_read     |  bool     | `false`          | `false ` | If `true` the object will be deleted after being rehydrated.                                                                                                                             |
| role_arn           |  string   |                  | `false ` | The Role ARN to be assumed, this will be used over credentials if specified.                                                                                                                             |
| poll_interval      |  string   | `1m`             | `false ` | How often to read a new set of objects. This value is mostly to control how often the object API is called to ensure once rehydration is done the receiver isn't making too many API calls. |
| poll_timeout       |  string   | `30s`            | `false ` | The timeout used when reading objects from AWS. |
| storage            |  string   |                  | `false ` | The component ID of a storage extension. The storage extension prevents duplication of data after a collector restart by remembering which objects were previously rehydrated.           |

## AWS Credential Configuration

Credentials are not configured in the receiver but rather in the environment.

Follow the [guidelines](https://aws.github.io/aws-sdk-go-v2/docs/configuring-sdk/#specifying-credentials) for the
credential configuration.

## Example Configuration

### Basic Configuration

This configuration specifies a `region`, `s3_bucket`, `starting_time`, and `ending_time`. 
This will rehydrate all objects in the bucket `my-bucket` that have a path that represents they were created between `1:00pm` and `2:30pm` UTC time on `October 1, 2023`.

Such a path could look like the following:
```
year=2023/month=10/day=01/hour=13/minute=30/metrics_12345.json
year=2023/month=10/day=01/hour=13/minute=30/logs_12345.json
year=2023/month=10/day=01/hour=13/minute=30/traces_12345.json
```

```yaml
awss3rehydration:
    region: "us-east-2"
    s3_bucket: "my-bucket"
    starting_time: 2023-10-01T13:00
    ending_time: 2023-10-01T14:30
```

### Using Storage Extension Configuration

This configuration shows using a storage extension to track rehydration progress over agent restarts. The `storage` field is set to the component ID of the storage extension.


```yaml
extensions:
    file_storage:
      directory: $OIQ_OTEL_COLLECTOR_HOME/storage

receivers:
    awss3rehydration:
        region: "us-east-2"
        s3_bucket: "my-bucket"
        starting_time: 2023-10-01T13:00
        ending_time: 2023-10-01T14:30
        storage: "file_storage"
```

### Root Folder Configuration

This configuration specifies an additional field `s3_prefix` to match the `s3_prefix` value of the AWS S3 Exporter. 
The `s3_prefix` value in the exporter will prefix the object path with the root folder and it needs to be accounted for in the rehydration receiver.

Such a path could look like the following:
```
root/year=2023/month=10/day=01/hour=13/minute=30/metrics_12345.json
root/year=2023/month=10/day=01/hour=13/minute=30/logs_12345.json
root/year=2023/month=10/day=01/hour=13/minute=30/traces_12345.json
```

```yaml
awss3rehydration:
    region: "us-east-2"
    s3_bucket: "my-bucket"
    starting_time: 2023-10-01T13:00
    ending_time: 2023-10-01T14:30
    s3_prefix: "root"
```

### Delete on read Configuration

This configuration enables the `delete_on_read` functionality which will delete an object from AWS after it has been successfully rehydrated into OTLP data and sent onto the next component in the pipeline. 

```yaml
awss3rehydration:
    region: "us-east-2"
    s3_bucket: "my-bucket"
    starting_time: 2023-10-01T13:00
    ending_time: 2023-10-01T14:30
    delete_on_read: true
```
