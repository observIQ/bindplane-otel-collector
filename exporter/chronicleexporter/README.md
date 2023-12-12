# Chronicle Exporter

This exporter facilitates the sending of logs to Chronicle, which is a security analytics platform provided by Google. It is designed to integrate with OpenTelemetry collectors to export telemetry data such as logs to a Chronicle account.

## Minimum Collector Versions

- Introduced: [v1.39.0](https://github.com/observIQ/bindplane-agent/releases/tag/v1.39.0)

## Supported Pipelines

- Logs

## How It Works

1. The exporter uses the configured credentials to authenticate with the Google Cloud services.
2. It marshals logs into the format expected by Chronicle.
3. It sends the logs to the appropriate regional Chronicle endpoint.

## Configuration

The exporter can be configured using the following fields:

| Field               | Type   | Default | Required | Description                                                                                                                                                           |
| ------------------- | ------ | ------- | -------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `region`            | string |         | `false`  | The region where the data will be sent, it must be one of the predefined regions. if no region is specfied defaults to `https://malachiteingestion-pa.googleapis.com` |
| `creds_file_path`   | string |         | `true`   | The file path to the Google credentials JSON file.                                                                                                                    |
| `creds`             | string |         | `true`   | The Google credentials JSON.                                                                                                                                          |
| `log_type`          | string |         | `true`   | The type of log that will be sent.                                                                                                                                    |
| `raw_log_field`     | string |         | `false`  | The field name for raw logs.                                                                                                                                          |
| `customer_id`       | string |         | `false`  | The customer ID used for sending logs.                                                                                                                                |
| `override_log_type` | bool   | `true`  | `false`  | Whether or not to override the `log_type` in the config with `attributes["log_type"]`                                                                                 |
| `namespace`         | string |         | `false`  | User-configured environment namespace to identify the data domain the logs originated from.                                                                           |

### Regions

Predefined regions include multiple global locations such as `Europe Multi-Region`, `Frankfurt`, `London`, `Singapore`, `Sydney`, `Tel Aviv`, `United States Multi-Region`, and `Zurich`. Each region has a specific endpoint URL.

### Log Type

If the `attributes["log_type"]` field is present in the log, and maps to a known Chronicle `log_type` the exporter will use the value of that field as the log type. If the `attributes["log_type"]` field is not present, the exporter will use the value of the `log_type` configuration field as the log type.

## Credentials

This exporter requires a Google Cloud service account with access to the Chronicle API. The service account must have access to the following endpoint(s):

The base endpoint is `https://malachiteingestion-pa.googleapis.com`

Alternatively, if a `region` is specified:

| Region                       | Endpoint                                                            |
| ---------------------------- | ------------------------------------------------------------------- |
| `Europe Multi-Region`        | `https://malachiteingestion-pa-europe.googleapis.com`               |
| `Frankfurt`                  | `https://malachiteingestion-pa-europe-west3.googleapis.com`         |
| `London`                     | `https://malachiteingestion-pa-europe-west2.googleapis.com`         |
| `Singapore`                  | `https://malachiteingestion-pa-asia-southeast1.googleapis.com`      |
| `Sydney`                     | `https://malachiteingestion-pa-australia-southeast1.googleapis.com` |
| `Tel Aviv`                   | `https://malachiteingestion-pa-europe-west4.googleapis.com`         |
| `United States Multi-Region` | `https://malachiteingestion-pa.googleapis.com`                      |
| `Zurich`                     | `https://malachiteingestion-pa-europe-west6.googleapis.com`         |

For additional information on accessing Chronicle, see the [Chronicle documentation](https://cloud.google.com/chronicle/docs/reference/ingestion-api#getting_api_authentication_credentials).

## Example Configuration

```yaml
chronicle:
  region: "Europe Multi-Region"
  creds_file_path: "/path/to/google/creds.json"
  log_type: "threat_detection"
  customer_id: "customer-123"
```
