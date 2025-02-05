# Getting Started

OpenTelemetry is at the core of standardizing telemetry solutions. At observIQ, we’re focused on building the very best in open source telemetry software. Our relationship with OpenTelemetry began in 2021, with observIQ, contributing our logging agent, Stanza, to the OpenTelemetry community. Now, we are shifting our focus to simplifying OpenTelemetry solutions to its large base of users. On that note, we launched a collector that combines the best of both worlds, with OpenTelemetry at its core, combined with observIQ’s functionalities to simplify its usage.

In this post, we are taking you through the installation of the Bindplane Agent and the steps to configure the agent to gather host metrics, eventually forwarding those metrics to the Google Cloud Operations.

## Installing the agent

The simplest way to get started is with one of the single-line installation commands shown below. For more advanced options, you'll find a variety of installation options for Linux, Windows, and macOS on GitHub.

Use the following single-line installation script to install the Bindplane Agent.
Please note that the agent must be installed on the system which you wish to collect host metrics from.

#### Windows:

```batch
msiexec /i "https://github.com/observIQ/bindplane-otel-collector/releases/latest/download/observiq-otel-collector.msi" /quiet
```

#### Linux:

```shell
sudo sh -c "$(curl -fsSlL https://github.com/observIQ/bindplane-otel-collector/releases/latest/download/install_unix.sh)" install_unix.sh
```

For more details on installation, see our [Linux](/docs/installation-linux.md), [Windows](/docs/installation-windows.md), and [Mac](/docs/installation-mac.md) installation guides.

## Setting up pre-requisites and authentication credentials

In the following example, we are using Google Cloud Operations as the destination. However, OpenTelemetry offers exporters for many destinations. Check out the list of exporters [here](/docs/exporters.md). 

### Setting up Google Cloud exporter prerequisites:

If running outside of Google Cloud (On prem, AWS, etc) or without the Cloud Monitoring scope, the Google Exporter requires a service account.
Create a service account with the following roles:

Metrics: `roles/monitoring.metricWriter`

Logs: `roles/logging.logWriter`

Create a service account JSON key and place it on the system that is running the collector.

### Linux

In this example, the key is placed at `/opt/observiq-otel-collector/sa.json` and its permissions are restricted to the user running the collector process.

```shell
sudo cp sa.json /opt/observiq-otel-collector/sa.json
sudo chown observiq-otel-collector: /opt/observiq-otel-collector/sa.json
sudo chmod 0400 /opt/observiq-otel-collector/sa.json
```
 
Set the `GOOGLE_APPLICATION_CREDENTIALS` environment variable by creating a systemd override. A systemd override allows users to modify the systemd service configuration without modifying the service directly. This allows package upgrades to happen seamlessly. You can learn more about systemd units and overrides here.

Run the following command

```shell
sudo systemctl edit observiq-otel-collector
```

If this is the first time an override is being created, paste the following contents into the file:

```
[Service]
Environment=GOOGLE_APPLICATION_CREDENTIALS=/opt/observiq-otel-collector/sa.json
```
 
If an override is already in place, simply insert the Environment parameter into the existing Service section.

Reload Systemd:

```shell
sudo systemctl daemon-reload
```

Restart the agent

```shell
sudo systemctl restart observiq-otel-collector
```
 
### Windows

In this example, the key is placed at `C:/observiq/collector/sa.json`.
Set the `GOOGLE_APPLICATION_CREDENTIALS` with the command prompt setx command.

Run the following command

```batch
setx GOOGLE_APPLICATION_CREDENTIALS "C:/observiq/collector/sa.json" /m
```
 
Restart the service using the services application.

## Configuring the agent

In this sample configuration, the steps to use the host metrics receiver to fetch metrics from the host system and export them to Google Cloud Operations are detailed. This is how it works:

The agent scrapes metrics and logs from the host and exports them to a destination assigned in the configuration file. 
To export the metrics to Google Cloud Operations, use the configurations outlined in the googlecloudexporter as in the example `config.yaml` below.

After the installation, the config file for the agent can be found at:

Windows: `C:\Program Files\observIQ OpenTelemetry Collector\config.yaml`

Linux: `/opt/observiq-otel-collector/config.yaml`

Edit the configuration file and use the following configuration.

```yaml
# Receivers collect metrics from a source. The host metrics receiver will
# get CPU load metrics about the machine the agent is running on
# every minute.
receivers:
  hostmetrics:
    collection_interval: 60s
    scrapers:
      cpu:
      disk:
      load:
      filesystem:
      memory:
      network:
      paging:
      processes:

# Exporters send the data to a destination, in this case GCP.
exporters:
  googlecloud:

# Service specifies how to construct the data pipelines using
# the configurations above.
service:
  pipelines:
    metrics:
      receivers: [hostmetrics]
      exporters: [googlecloud]
```

Restart the agent

```shell
systemctl restart observiq-otel-collector
```

## Viewing the metrics in Google Cloud Operations

You should now be able to view the host metrics in your Metrics explorer. Nice work! This is how simple it is to collect host metrics with the Bindplane Agent.

### Metrics collected

| Metric | Description | Metric Namespace |
| --- | --- | --- |
| Processes Created | Total number of created processes. | custom.googleapis.com/opencensus/system.processes.created |
| Process Count | Total number of processes in each state. | custom.googleapis.com/opencensus/system.processes.count |
| Process CPU time | Total CPU seconds broken down by different states. | custom.googleapis.com/opencensus/process.cpu.time |
| Process Disk IO | Disk bytes transferred. | custom.googleapis.com/opencensus/process.disk.io |
| File System Inodes Used | FileSystem inodes used. | custom.googleapis.com/opencensus/system.filesystem.inodes.usage |
| File System Utilization | Filesystem bytes used. | custom.googleapis.com/opencensus/system.filesystem.usage |
| Process Physical Memory Utilization | The amount of physical memory in use. | custom.googleapis.com/opencensus/process.memory.physical_usage |
| Process Virtual Memory Utilization | Virtual memory size. | custom.googleapis.com/opencensus/process.memory.virtual_usage |
| Networking Errors | The number of errors encountered. | custom.googleapis.com/opencensus/system.network.errors |
| Networking Connections | The number of connections. | custom.googleapis.com/opencensus/system.network.connections |

## What Next?

Check out our list of supported [receivers](), [processors](), [exporters](), and [extensions]() for more information about making a config. To see more monitoring examples, be sure to follow the [Observability Blog](https://observiq.com/blog/).

observIQ’s distribution is a game-changer for companies looking to implement the OpenTelemetry standards. The single line installer, seamlessly integrated receivers, exporter, and processor pool make working with this agent simple. For questions, requests, and suggestions, reach out to our support team at support@observIQ.com.
