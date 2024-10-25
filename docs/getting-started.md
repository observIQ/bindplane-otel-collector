# Getting Started

OpenTelemetry is at the core of standardizing telemetry solutions. At observIQ, we’re focused on building the very best in open source telemetry software. Our relationship with OpenTelemetry began in 2021, with observIQ, contributing our logging agent, Stanza, to the OpenTelemetry community. Now, we are shifting our focus to simplifying OpenTelemetry solutions to its large base of users. On that note, we launched a collector that combines the best of both worlds, with OpenTelemetry at its core, combined with observIQ’s functionalities to simplify its usage.

In this post, we are taking you through the installation of the BindPlane Agent and the steps to configure the agent to gather host metrics, eventually forwarding those metrics to the Google Cloud Operations.

## Connect to BindPlane

We'll need to provide the installation scripts with an OpAMP management endpoint and a secret key so that we can pass a configuration to the supervisor. Access BindPlane for these values.

## Installing the agent

The simplest way to get started is with one of the installation commands shown below. For more advanced options, you'll find a variety of installation options for Linux, Windows, and macOS on GitHub.

Please note that the agent must be installed on the system which you wish to collect host metrics from.

#### Windows:

```pwsh
msiexec /i "https://github.com/observIQ/bindplane-otel-collector/releases/latest/download/observiq-otel-collector.msi" /quiet ENABLEMANAGEMENT="1" OPAMPENDPOINT="<your_endpoint>" OPAMPSECRETKEY="<your_secret_key>"
```

#### Linux:

```shell
sudo sh -c "$(curl -fsSlL https://github.com/observIQ/bindplane-otel-collector/releases/latest/download/install_unix.sh)" install_unix.sh -e '<your_endpoint>' -s '<your_secret_key>'
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

Restart the supervisor

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

With the agent installed it should be connected to your BindPlane instance. Navigate to BindPlane and create a configuration with the "Host Metrics" source and "Google Cloud Platform" destination.

Once the configuration is made, roll it out to this agent to begin collecting data and sending to GCP.

## Viewing the metrics in Google Cloud Operations

You should now be able to view the host metrics in your Metrics explorer. Nice work! This is how simple it is to collect host metrics with the BindPlane Agent.

### Metrics collected

| Metric                              | Description                                        | Metric Namespace                                                |
| ----------------------------------- | -------------------------------------------------- | --------------------------------------------------------------- |
| Processes Created                   | Total number of created processes.                 | custom.googleapis.com/opencensus/system.processes.created       |
| Process Count                       | Total number of processes in each state.           | custom.googleapis.com/opencensus/system.processes.count         |
| Process CPU time                    | Total CPU seconds broken down by different states. | custom.googleapis.com/opencensus/process.cpu.time               |
| Process Disk IO                     | Disk bytes transferred.                            | custom.googleapis.com/opencensus/process.disk.io                |
| File System Inodes Used             | FileSystem inodes used.                            | custom.googleapis.com/opencensus/system.filesystem.inodes.usage |
| File System Utilization             | Filesystem bytes used.                             | custom.googleapis.com/opencensus/system.filesystem.usage        |
| Process Physical Memory Utilization | The amount of physical memory in use.              | custom.googleapis.com/opencensus/process.memory.physical_usage  |
| Process Virtual Memory Utilization  | Virtual memory size.                               | custom.googleapis.com/opencensus/process.memory.virtual_usage   |
| Networking Errors                   | The number of errors encountered.                  | custom.googleapis.com/opencensus/system.network.errors          |
| Networking Connections              | The number of connections.                         | custom.googleapis.com/opencensus/system.network.connections     |

## What Next?

Check out our list of supported [receivers](), [processors](), [exporters](), and [extensions]() for more information about making a config. To see more monitoring examples, be sure to follow the [Observability Blog](https://observiq.com/blog/).

observIQ’s distribution is a game-changer for companies looking to implement the OpenTelemetry standards. The single line installer, seamlessly integrated receivers, exporter, and processor pool make working with this agent simple. For questions, requests, and suggestions, reach out to our support team at support@observIQ.com.
