# Google Cloud Storage Exporter

This exporter allows you to export logs, metrics, and traces to Google Cloud Storage. Telemetry is exported in [OpenTelemetry Protocol JSON format](https://github.com/open-telemetry/opentelemetry-proto).

## Minimum Agent Versions

- Introduced: [v1.72.0](https://github.com/observIQ/bindplane-otel-collector/releases/tag/v1.72.0)

## Supported Pipelines

- Logs
- Metrics
- Traces

## How It Works

1. The exporter marshals any telemetry sent to it into OTLP Json Format. It will apply compression if configured to do so.
2. Exports telemetry to Google Cloud Storage. See [Object Path](#object-path) for more information on the expected object path format.

## Notes

1. The exporter will create a bucket if it doesn't exist. Bucket names in GCS are _globally_ unique, so if any organization contains a bucket with the given name, it will fail to create and will likely give 403 Forbidden codes back when it tries to write. You can manually create the bucket in GCS to ensure the name is not taken. [More info](https://cloud.google.com/storage/docs/buckets)
2. You can authenticate to Google Cloud using the provided `credentials`, `credentials_file`, or by using Application Default Credentials.

## Configuration

| Field                | Type   | Default  | Required | Description                                                                                                         |
| -------------------- | ------ | -------- | -------- | ------------------------------------------------------------------------------------------------------------------- |
| project_id           | string |          | `true`   | The ID of the Google Cloud project the bucket belongs to.                                                           |
| bucket_name          | string |          | `true`   | The name of the bucket to store objects in.                                                                         |
| bucket_location      | string |          | `false`  | The location of the bucket. Uses GCS default location if not set. Can only be set during bucket creation.           |
| bucket_storage_class | string |          | `false`  | The storage class of the bucket. Uses GCS default storage class if not set. Can only be set during bucket creation. |
| folder_name          | string |          | `false`  | An optional folder to put the objects in.                                                                           |
| object_prefix        | string |          | `false`  | An optional prefix to prepend to the object file name.                                                              |
| credentials          | string |          | `false`  | Optional credentials to provide authentication to Google Cloud. Mutually exclusive with `credentials_file`.         |
| credentials_file     | string |          | `false`  | Optional file path to credentials to provide authentication to Google Cloud. Mutually exclusive with `credentials`. |
| partition            | string | `minute` | `false`  | Time granularity of object name. Valid values are `hour` or `minute`.                                               |
| compression          | string | `none`   | `false`  | The type of compression applied to the data before sending it to storage. Valid values are `none` and `gzip`.       |

### Object Path

Object paths will be in the form:

```
{folder_name}/year=XXXX/month=XX/day=XX/hour=XX/minute=XX
```

## Example Configurations

### Minimal Configuration

This configuration only specifies a project ID and the bucket name. The exporter will use the default partition of `minute` and will not apply compression. It will use the Google Cloud default location and storage class for the bucket, will not add a folder name or object prefix, and will authenticate to google cloud using application default credentials instead of `credentials` or `credentials_file`.

```yaml
googlecloudstorage:
  project_id: "my-project-id-18352"
  bucket_name: "my-bucket-name"
```

Example Object Names:

```
year=2021/month=01/day=01/hour=01/minute=00/metrics_{random_id}.json
year=2021/month=01/day=01/hour=01/minute=00/logs_{random_id}.json
year=2021/month=01/day=01/hour=01/minute=00/traces_{random_id}.json
```

### Hour Partition Configuration with Compression

This shows specifying a partition of `hour` and that the `minute=XX` portion of the blob path will be omitted. It also adds compression, reducing the object size significantly.

```yaml
googlecloudstorage:
  project_id: "my-project-id-18352"
  bucket_name: "my-bucket-name"
  partition: "hour"
  compression: "gzip"
```

Example Object Names:

```
year=2021/month=01/day=01/hour=01/metrics_{random_id}.json.gz
year=2021/month=01/day=01/hour=01/logs_{random_id}.json.gz
year=2021/month=01/day=01/hour=01/traces_{random_id}.json.gz
```

### Full Configuration

This configuration shows all fields filled out.

```yaml
googlecloudstorage:
  project_id: "my-project-id-18352"
  bucket_name: "my-bucket-name"
  bucket_location: "US-EAST1"
  bucket_storage_class: "NEARLINE"
  credentials_file: "/path/to/googlecloud/credentials/file"
  folder_name: "my-folder-name"
  object_prefix: "object-prefix_"
  partition: "hour"
  compression: "gzip"
```

Example Blob Names:

```
my-folder-name/year=2021/month=01/day=01/hour=01/object-prefix_metrics_{random_id}.json.gz
my-folder-name/year=2021/month=01/day=01/hour=01/object-prefix_logs_{random_id}.json.gz
my-folder-name/year=2021/month=01/day=01/hour=01/object-prefix_traces_{random_id}.json.gz
```
