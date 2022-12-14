# SAP Netweaver Receiver
This receiver collects metrics from SAP Netweaver instance based on the [SAPControl Web Service Interface](https://www.sap.com/documents/2016/09/0a40e60d-8b7c-0010-82c7-eda71af511fa.html). Stats are collected from the `GetAlertTree` and `EnqGetLockTable` methods.

## Supported Pipelines
- Metrics

## How It Works
1. The user configures this receiver in a pipeline.
2. The user configures a supported component to route telemetry from this receiver.

## Prerequisites
- SAP Netweaver 7.10+
- The ability to authenticate to the OS
- SAP read-only permission
- For SAP version 7.38+,  (Windows: execute permission, Unix: write permission) is required for each instance of sapstartsrv on the host. If this permission is being assigned to a group, the - monitoring user in the group must have the group set as primary. If authentication or authorization check fails the request will fail with “Invalid Credentials" or "Permission denied" fault string.

More information on how to setup a SAP NetWeaver Stack for each operating system and version can be found [here](https://help.sap.com/docs/SAP_NETWEAVER/9e41ead9f54e44c1ae1a1094b0f80712/576f5c1808de4d1abecbd6e503c9ba42.html?language=en-US).

## Configuration
| Field               | Type               | Default                                                                                   | Description                                                                                                                                                  |
|---------------------|--------------------|-------------------------------------------------------------------------------------------|--------------------------------------------------------------------------------------------------------------------------------------------------------------|
| metrics             | map             | (default: see `DefaultMetricsSettings` [here](./internal/metadata/generated_metrics.go) | Allows enabling and disabling [specific metrics](./documentation.md#metrics) from being collected in this receiver.                                                                        |
| endpoint            | string             | `http://localhost:50013`                                                                  | The name of the metric created.                                                                                                                              |
| username            | string             | `(no default)`                                                                            | Specifies the username used to authenticate using basic auth.                                                                                                |
| password            | string             | `(no default)`                                                                            | Specifies the password used to authenticate using basic auth.                                                                                                |
| collection_interval | duration | `60s`                                                                           | This receiver collects metrics on an interval. This value must be a string readable by Golang's [time.ParseDuration](https://pkg.go.dev/time#ParseDuration). |

### Example Configuration
```yaml
receivers:
  sapnetweaver:
    metrics:
    endpoint: http://localhost:50013
    username: root
    password: password
    collection_interval: 60s
processors:
    batch:
exporters:
    googlecloud:

service:
    pipelines:
        metrics:
            receivers: [sapnetweaver]
            processors: [batch]
            exporters: [googlecloud]
```

The full list of settings exposed for this receiver are documented [here](./config.go) with detailed sample configurations [here](./testdata/config.yaml).

## Metrics
Details about the metrics produced by this receiver can be found in [metadata.yaml](./metadata.yaml)
