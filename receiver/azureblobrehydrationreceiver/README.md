# Azure Blob Storage Rehydration Receiver
Rehydrates OTLP from Azure Blob Storage that was stored using the Azure Blob Exporter [../../exporter/azureblobexporter/README.md].

## Important Note
This is not a traditional receiver that continually produces data but rather rehydrates all blobs found within a specified time range. Once all of the blobs have been rehydrated in that time range the receiver will stop producing data. After the receiver has detected three consecutive empty polls it will stop polling for new blobs in order to prevent unnecessary API calls.

## Minimum Agent Versions
- Introduced: [v1.37.0](https://github.com/observIQ/bindplane-agent/releases/tag/v1.37.0)

## Supported Pipelines
- Metrics
- Logs
- Traces

## How it works
1. The receiver polls blob storage for all blobs in the specified container.
2. The receiver will parse each blob's path to determine if it matches a path created by the [Azure Blob Exporter](../../exporter/azureblobexporter/README.md#blob-path).
3. If the blob path is from the exporter, the receiver will parse the timestamp represented by the path.
4. If the timestamp is within the configured range the receiver will download the blob and parse its contents into OTLP data.

    a. The receiver will process both uncompressed JSON blobs and blobs compressed with gzip.

## Configuration
| Field              | Type      | Default          | Required | Description                                                                                                                                                                            |
|--------------------|-----------|------------------|----------|----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| connection_string  |  string   |                  | `true`   | The connection string to the Azure Blob Storage account. Can be found under the `Access keys` section of your storage account.                                                         |
| container          |  string   |                  | `true`   | The name of the container to rehydrate from.                                                                                                                                           |
| root_folder        |  string   |                  | `false`  | The root folder that prefixes the blob path. Should match the `root_folder` value of the Azure Blob Exporter.                                                                          |
| starting_time      |  string   |                  | `true `  | The UTC start time that represents the start of the time range to rehydrate from. Must be in the form `YYYY-MM-DDTHH:MM`.                                                              |
| ending_time        |  string   |                  | `true `  | The UTC end time that represents the end of the time range to rehydrate from. Must be in the form `YYYY-MM-DDTHH:MM`.                                                                  |
| delete_on_read     |  bool     | `false`          | `false ` | If `true` the blob will be deleted after being rehydrated.                                                                                                                             |
| poll_interval      |  string   | `1m`             | `false ` | How often to read a new set of blobs. This value is mostly to control how often the blob API is called to ensure once rehydration is done the receiver isn't making too many API calls. |
| storage            |  string   |                  | `false ` | The component ID of a storage extension. The storage extension prevents duplication of data after a collector restart by remembering which blobs were previously rehydrated.           |

## Example Configuration

### Basic Configuration

This configuration specifies a `connection_string`, `container`, `starting_time`, and `ending_time`. 
This will rehydrate all blobs in the container `my-container` that have a path that represents they were created between `1:00pm` and `2:30pm` UTC time on `October 1, 2023`.

Such a path could look like the following:
```
year=2023/month=10/day=01/hour=13/minute=30/metrics_12345.json
year=2023/month=10/day=01/hour=13/minute=30/logs_12345.json
year=2023/month=10/day=01/hour=13/minute=30/traces_12345.json
```

```yaml
azureblobrehydration:
    connection_string: "DefaultEndpointsProtocol=https;AccountName=storage_account_name;AccountKey=storage_account_key;EndpointSuffix=core.windows.net"
    container: "my-container"
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
    azureblobrehydration:
        connection_string: "DefaultEndpointsProtocol=https;AccountName=storage_account_name;AccountKey=storage_account_key;EndpointSuffix=core.windows.net"
        container: "my-container"
        starting_time: 2023-10-01T13:00
        ending_time: 2023-10-01T14:30
        storage: "file_storage"
```

### Root Folder Configuration

This configuration specifies an additional field `root_folder` to match the `root_folder` value of the Azure Blob Exporter. 
The `root_folder` value in the exporter will prefix the blob path with the root folder and it needs to be accounted for in the rehydration receiver.

Such a path could look like the following:
```
root/year=2023/month=10/day=01/hour=13/minute=30/metrics_12345.json
root/year=2023/month=10/day=01/hour=13/minute=30/logs_12345.json
root/year=2023/month=10/day=01/hour=13/minute=30/traces_12345.json
```

```yaml
azureblobrehydration:
    connection_string: "DefaultEndpointsProtocol=https;AccountName=storage_account_name;AccountKey=storage_account_key;EndpointSuffix=core.windows.net"
    container: "my-container"
    starting_time: 2023-10-01T13:00
    ending_time: 2023-10-01T14:30
    root_folder: "root"
```

### Delete on read Configuration

This configuration enables the `delete_on_read` functionality which will delete a blob from Azure after it has been successfully rehydrated into OTLP data and sent onto the next component in the pipeline. 

```yaml
azureblobrehydration:
    connection_string: "DefaultEndpointsProtocol=https;AccountName=storage_account_name;AccountKey=storage_account_key;EndpointSuffix=core.windows.net"
    container: "my-container"
    starting_time: 2023-10-01T13:00
    ending_time: 2023-10-01T14:30
    delete_on_read: true
```
