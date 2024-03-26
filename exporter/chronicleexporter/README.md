# Chronicle Exporter

**Currently only v2 of the ingestion API is supported**

This exporter facilitates the sending of logs to Chronicle, which is a security analytics platform provided by Google. It is designed to integrate with OpenTelemetry collectors to export telemetry data such as logs to a Chronicle account.

## Minimum Collector Versions

- Introduced: [v1.39.0](https://github.com/observIQ/bindplane-agent/releases/tag/v1.39.0)

## Supported Pipelines

- Logs

## How It Works

1. The exporter uses the configured credentials to authenticate with the Google Cloud services.
2. It marshals logs into the format expected by Chronicle.
3. It sends the logs to the appropriate Chronicle endpoint.

## Configuration

The exporter can be configured using the following fields:

| Field               | Type              | Default                                        | Required | Description                                                                                 |
| ------------------- | ----------------- | ---------------------------------------------- | -------- | ------------------------------------------------------------------------------------------- |
| `endpoint`          | string            | `https://malachiteingestion-pa.googleapis.com` | `false`  | The Endpoint for sending to chronicle.                                                      |
| `creds_file_path`   | string            |                                                | `true`   | The file path to the Google credentials JSON file.                                          |
| `creds`             | string            |                                                | `true`   | The Google credentials JSON.                                                                |
| `log_type`          | string            |                                                | `true`   | The type of log that will be sent.                                                          |
| `raw_log_field`     | string            |                                                | `false`  | The field name for raw logs.                                                                |
| `customer_id`       | string            |                                                | `false`  | The customer ID used for sending logs.                                                      |
| `override_log_type` | bool              | `true`                                         | `false`  | Whether or not to override the `log_type` in the config with `attributes["log_type"]`       |
| `namespace`         | string            |                                                | `false`  | User-configured environment namespace to identify the data domain the logs originated from. |
| `compression`       | string            | `none`                                         | `false`  | The compression type to use when sending logs. valid values are `none` and `gzip`           |
| `ingestion_labels`  | map[string]string |                                                | `false`  | Key-value pairs of labels to be applied to the logs when sent to chronicle.                 |

### Log Type

If the `attributes["log_type"]` field is present in the log, and maps to a known Chronicle `log_type` the exporter will use the value of that field as the log type. If the `attributes["log_type"]` field is not present, the exporter will use the value of the `log_type` configuration field as the log type.

currently supported log types are:

- windows_event.security
- windows_event.custom
- windows_event.application
- windows_event.system
- sql_server

## Credentials

This exporter requires a Google Cloud service account with access to the Chronicle API. The service account must have access to the endpoint specfied in the config.
Besides the default endpoint, there are also regional endpoints that can be used [here](https://cloud.google.com/chronicle/docs/reference/ingestion-api#regional_endpoints).

For additional information on accessing Chronicle, see the [Chronicle documentation](https://cloud.google.com/chronicle/docs/reference/ingestion-api#getting_api_authentication_credentials).

## Example Configuration

### Basic Configuration

```yaml
chronicle:
  creds_file_path: "/path/to/google/creds.json"
  log_type: "ABSOLUTE"
  customer_id: "customer-123"
```

### Basic Configuration with Regional Endpoint

```yaml
chronicle:
  endpoint: https://malachiteingestion-pa.googleapis.com
  creds_file_path: "/path/to/google/creds.json"
  log_type: "ONEPASSWORD"
  customer_id: "customer-123"
```

### Configuration with Ingestion Labels

```yaml
chronicle:
  creds_file_path: "/path/to/google/creds.json"
  log_type: ""
  customer_id: "customer-123"
  ingestion_labels: 
    env: dev
    zone: USA
```