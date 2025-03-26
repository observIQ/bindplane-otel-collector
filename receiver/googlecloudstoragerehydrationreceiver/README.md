# Google Cloud Storage Rehydration Receiver

This receiver rehydrates OTLP data from Google Cloud Storage that was previously stored using the [Google Cloud Storage Exporter](../../exporter/googlecloudstorageexporter/README.md).

## Important Note

This is not a traditional receiver that continually produces data. Instead, it rehydrates all objects found within a specified time range. Once all objects have been rehydrated in that time range, the receiver will stop producing data.

## Minimum Agent Versions

- Introduced: [v1.74.0](https://github.com/observIQ/bindplane-otel-collector/releases/tag/v1.74.0)

## Supported Pipelines

- Metrics
- Logs
- Traces

## How it works

1. The receiver streams objects from the specified GCS bucket
2. The receiver parses each object's path to determine if it matches the path format created by the [Google Cloud Storage Exporter](../../exporter/googlecloudstorageexporter/README.md#object-path).
3. If the object path is from the exporter, the receiver parses the timestamp represented by the path.
4. If the timestamp is within the configured range, the receiver downloads the object and parses its contents into OTLP data.
   - The receiver can process both uncompressed JSON objects and objects compressed with gzip.

> Note: Objects outside the time range are still retrieved and then filtered via the `starting_time` and `ending_time` values.

## Notes

1. You can authenticate to Google Cloud using the provided `credentials`, `credentials_file`, or by using Application Default Credentials.
2. Your authentication credentials must have the Storage Admin permission to read and delete objects.

## Configuration

| Field            | Type   | Default | Required | Description                                                                                                                                                                    |
| ---------------- | ------ | ------- | -------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| bucket_name      | string |         | `true`   | The name of the bucket to rehydrate from.                                                                                                                                      |
| project_id       | string |         | `false`  | The ID of the Google Cloud project the bucket belongs to. Will be read from credentials if not set.                                                                            |
| folder_name      | string |         | `false`  | The folder within the bucket to rehydrate from. Should match the `folder_name` value of the Google Cloud Storage Exporter.                                                     |
| starting_time    | string |         | `true`   | The UTC start time that represents the start of the time range to rehydrate from. Must be in the form `YYYY-MM-DDTHH:MM`.                                                      |
| ending_time      | string |         | `true`   | The UTC end time that represents the end of the time range to rehydrate from. Must be in the form `YYYY-MM-DDTHH:MM`.                                                          |
| delete_on_read   | bool   | `false` | `false`  | If `true`, the object will be deleted after being rehydrated.                                                                                                                  |
| storage          | string |         | `false`  | The component ID of a storage extension. The storage extension prevents duplication of data after a collector restart by remembering which objects were previously rehydrated. |
| credentials      | string |         | `false`  | Optional credentials to provide authentication to Google Cloud.                                                                                                                |
| credentials_file | string |         | `false`  | Optional file path to credentials to provide authentication to Google Cloud.                                                                                                   |
| batch_size       | int    | `30`    | `false`  | The number of objects to download and process in the pipeline simultaneously. This parameter directly impacts performance by controlling the concurrent object download limit. |

## Example Configurations

### Basic Configuration

This configuration specifies a `bucket_name`, `starting_time`, and `ending_time`.
This will rehydrate all objects in the bucket `my-bucket` that have a path representing they were created between `1:00pm` and `2:30pm` UTC time on `October 1, 2023`.

Object paths in this range would look similar to the following:

```
year=2023/month=10/day=01/hour=13/minute=30/metrics_12345.json
year=2023/month=10/day=01/hour=13/minute=30/logs_12345.json
year=2023/month=10/day=01/hour=13/minute=30/traces_12345.json
```

```yaml
googlecloudstoragerehydration:
  bucket_name: "my-bucket"
  starting_time: "2023-10-01T13:00"
  ending_time: "2023-10-01T14:30"
```

### Using Storage Extension Configuration

This configuration shows using a storage extension to track rehydration progress over agent restarts. The `storage` field is set to the component ID of the storage extension.

```yaml
extensions:
  file_storage:
    directory: $OIQ_OTEL_COLLECTOR_HOME/storage
receivers:
  googlecloudstoragerehydration:
    bucket_name: "my-bucket"
    starting_time: "2023-10-01T13:00"
    ending_time: "2023-10-01T14:30"
    storage: "file_storage"
```

### Folder Configuration

This configuration specifies an additional field `folder_name` to limit the rehydration to only the objects within that folder.

Object paths would look similar to the following:

```
my-folder/year=2023/month=10/day=01/hour=13/minute=30/metrics_12345.json
my-folder/year=2023/month=10/day=01/hour=13/minute=30/logs_12345.json
my-folder/year=2023/month=10/day=01/hour=13/minute=30/traces_12345.json
```

```yaml
googlecloudstoragerehydration:
  bucket_name: "my-bucket"
  starting_time: "2023-10-01T13:00"
  ending_time: "2023-10-01T14:30"
  folder_name: "my-folder"
```

### Delete on Read Configuration

This configuration enables the `delete_on_read` functionality which will delete an object from GCS after it has been successfully rehydrated.

```yaml
googlecloudstoragerehydration:
  bucket_name: "my-bucket"
  starting_time: "2023-10-01T13:00"
  ending_time: "2023-10-01T14:30"
  delete_on_read: true
```

### Authentication Configuration

The previous examples assume Application Default Credentials are used to authenticate to Google Cloud. This configuration shows how to authenticate using credentials from a file:

```yaml
googlecloudstoragerehydration:
  bucket_name: "my-bucket"
  starting_time: "2023-10-01T13:00"
  ending_time: "2023-10-01T14:30"
  credentials_file: "/path/to/googlecloud/credentials/file"
```

Or using credentials directly:

```yaml
googlecloudstoragerehydration:
  bucket_name: "my-bucket"
  starting_time: "2023-10-01T13:00"
  ending_time: "2023-10-01T14:30"
  credentials: |
    {
      "type": "service_account",
      "project_id": "my-project",
      ...
    }
```
