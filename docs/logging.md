# Logging

The BDOT offers two methods of collecting logs. First is individual receivers such as Filelog and Journald. Second is the Plugin receiver which utilizes pre-configured plugins to gather logs from many different sources.

## Using Indivudal Receivers

To add logging using an individual receiver, add the receiver into your `config.yaml` similar to the Filelog example below. The available logging receivers include:

- [Filelog Receiver](https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/receiver/filelogreceiver)
- [TCP Log Receiver](https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/receiver/tcplogreceiver)
- [UDP Log Receiver](https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/receiver/udplogreceiver)
- [Syslog Receiver](https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/receiver/syslogreceiver)
- [Journald Receiver](https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/receiver/journaldreceiver)
- [Windows Events Receiver](https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/receiver/windowseventlogreceiver)

To see a full list of receivers, check the [Receivers](/docs/receivers.md) page.

The example below uses the Filelog receiver. Additional [operators](https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/main/pkg/stanza/docs/operators/README.md#what-operators-are-available) can be added to further parse logs. To see more details on the Filelog receiver, see the [OTel documentation](https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/receiver/filelogreceiver).

```yaml
receivers:
  filelog:
    # Add relevant log paths to the include parameter
    include: ["/tmp/*.log"]
    start_at: "end"
    # Optionally add operators for further parsing
    operators:
      - type: json_parser
        timestamp:
          parse_from: attributes.time
          layout: "%Y-%m-%d %H:%M:%S"

exporters:
  googlecloud:
    # To add logging to Google CLoud, add the log parameter to the exporter.
    project: my-project
    log:
      # Optionally, default_log_name parameter is not required
      default_log_name: opentelemetry.io/grpc-collector-exported-log

processors:
  batch:

service:
  pipelines:
    # Make sure to have a logs section in the pipeline
    logs:
      receivers: [filelog]
      processors: [batch]
      exporters: [googlecloud]
```

## Using the Plugin Receiver

To add logging using the Plugin receiver, add the receiver into your `config.yaml` similar to the example below. For more information on the Plugin receiver, see the [documentation page](/receiver/pluginreceiver/README.md). To see a full list of available plugins, see the [plugins folder](/plugins/).

```yaml
receivers:
  plugin:
    # Add the path to the plugin you wish to configure
    path: ./plugins/simplehost.yaml
    # Configure parameters for the specified plugin
    parameters:
      enable_cpu: false
      enable_memory: true

exporters:
  googlecloud:
    # To add logging to Google CLoud, add the log parameter to the exporter.
    project: my-project
    log:
      default_log_name: opentelemetry.io/grpc-collector-exported-log

processors:
  batch:

service:
  pipelines:
    # Make sure to have a logs section in the pipeline
    logs:
      receivers: [plugin]
      processors: [batch]
      exporters: [googlecloud]
```
