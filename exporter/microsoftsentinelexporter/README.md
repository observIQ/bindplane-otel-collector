# Microsoft Sentinel Exporter

This exporter allows you to export logs to Microsoft Sentinel via the Log Analytics Ingestion API. Logs are exported in [OpenTelemetry Protocol JSON format](https://github.com/open-telemetry/opentelemetry-proto).

## Minimum Agent Versions
- Introduced: 

## Supported Pipelines
- Logs

## How It Works


## Configuration
| Field              | Type      | Default          | Required | Description                                                                                                                    |
|--------------------|-----------|------------------|----------|--------------------------------------------------------------------------------------------------------------------------------|

## Example Configurations

### Minimal Configuration



```yaml

```

Example Blob Names:

```

```


```yaml
azureblob:
    connection_string: "DefaultEndpointsProtocol=https;AccountName=storage_account_name;AccountKey=storage_account_key;EndpointSuffix=core.windows.net"
    container: "my-container"
    partition: "hour"
```


### Full Configuration with compression


```yaml
```

