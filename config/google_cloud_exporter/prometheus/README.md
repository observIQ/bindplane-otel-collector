# Prometheus Metrics with Google Cloud

The Prometheus Receiver can be used to send prometheus style metrics scraped from prometheus exporters to Google Cloud Monitoring.

## Prerequisites

See the [prerequisites](../README.md) doc for Google Cloud prerequisites.

Ensure your applications support prometheus metrics, or have prometheus exporters configured.

## Usage

Modify the example configuration's `prometheus` receiver. Update
the `job_name` value to reflect your environment. Also update the
`targets` value(s) to point to your application's prometheus exporter(s).

For example, if you have five Nginx systems that need to be
monitored, your receiver configuration would look like this:

```yaml
receivers:
  prometheus:
    config:
      scrape_configs:
        - job_name: "nginx"
          scrape_interval: 60s
          static_configs:
            - targets:
                - "nginx-0:9113"
                - "nginx-1:9113"
                - "nginx-2:9113"
                - "nginx-3:9113"
                - "nginx-4:9113"
```

You can have multiple scrape configs, for targeting multiple applications.

```yaml
receivers:
  prometheus:
    config:
      scrape_configs:
        - job_name: "nginx"
          scrape_interval: 60s
          static_configs:
            - targets:
                - "nginx-0:9113"
        - job_name: "redis"
          scrape_interval: 60s
          static_configs:
            - targets:
                - "redis-0:9900"
```

### Deploy

1. Copy [config.yaml](./config.yaml) to `/opt/bindplane-otel-collector/config.yaml`
2. Modify the configuration to reflect your environment (See [Usage](./README.md#usage))
3. Restart the agent: `sudo systemctl restart bindplane-otel-collector`

You can search for metrics under the "Generic Node" section
with the prefix `workload.googleapis.com`.

### Metric Labels

| Label      | Description                                                                                                                                                                                                                                                                                                    | Example        |
| ---------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------------- |
| `node_id`  | The hostname of the agent. Set within the [Google exporter](https://github.com/observIQ/bindplane-otel-collector/tree/main/exporter/googlecloudexporter#metric-processing-steps), and required for [generic_node](https://cloud.google.com/monitoring/api/resources#tag_generic_node) monitored resource type. | `collector-0`  |
| `job`      | Dervied from the Prometheus receiver's `config.scrape_configs` `job_name` value. This value should represent the applications being scraped by the scrape config.                                                                                                                                              | `nodeexporter` |
| `instance` | The host / ip and port being scraped by the scrape config.                                                                                                                                                                                                                                                     | `node-1:9100`  |

For the best experience, ensure that the values for `job_name` and `targets` are descriptive, as the
`job` and `instance` metric labels are derived from those values.

### Custom Metric Prefix

Generally metric names will contain their software name, however, sometimes
you might have a metric that is not descriptive, such as `uptime`. In this case, you
can use a custom metric prefix.

```yaml
exporters:
  googlecloud/mycustomapp:
    metric:
      prefix: workload.googleapis.com/mycustomapp
...
service:
  pipelines:
    metrics:
      ...
      exporters:
      - googlecloud/mycustomapp
```

This configuration snipet contains a Google exporter dedicated to `mycustomapp`, which
has a metric prefix of `workload.googleapis.com/mycustomapp`.
