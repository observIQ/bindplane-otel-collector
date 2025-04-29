
## Docker Compose

The Bindplane Distro for OpenTelemetry (BDOT) Collector can be installed with Docker and Docker Compose.

## Prerequisites

Before installing the Bindplane Distro for OpenTelemetry Collector using Docker Compose, ensure you have the following installed:

- Docker Engine (version 20.10.0 or later)
- Docker Compose (version 2.0.0 or later)

## Installation Steps

1. Create directories and files to store both Docker Compose and BDOT Collector configuration files:

```
> config
    manager.yaml
> storage
    config.yaml
    logging.yaml
  docker-compose.yaml
```

Proceed to add content into all files except the `manager.yaml` that will be auto-updated when creating rolling out a config from Bindplane.

2. Paste the following content into your `docker-compose.yaml`:

```yaml
version: '3.8'

services:
  bdot-collector:
    image: ghcr.io/observiq/bindplane-agent:latest
    command: ["--config=/etc/otel/storage/config.yaml"]
    volumes:
      - ./config:/etc/otel/config
      - ./storage:/etc/otel/storage
    ports:
      - "4317:4317"   # OTLP gRPC
      - "4318:4318"   # OTLP HTTP
      - "13133:13133" # Health check extension
      - "55679:55679" # ZPages debugging
    environment:
      OPAMP_ENDPOINT: <your-endpoint> # use "wss://app.bindplane.com/v1/opamp" for Bindplane Cloud
      OPAMP_SECRET_KEY: <your-secret-key>
      OPAMP_LABELS: install_id=<your-install-id>
      OPAMP_AGENT_NAME: bdot-collector
      CONFIG_YAML_PATH: /etc/otel/storage/config.yaml
      MANAGER_YAML_PATH: /etc/otel/config/manager.yaml
      LOGGING_YAML_PATH: /etc/otel/storage/logging.yaml

```

Get your keys from the **Agents > Install Agents** page in Bindplane.

![Sample Config](assets/install-keys.png)

3. Paste this into your `config.yaml` file in the `storage` directory:

```yaml
receivers:
  nop:
processors:
  batch:
exporters:
  nop:
service:
  pipelines:
    metrics:
      receivers: [nop]
      processors: [batch]
      exporters: [nop]
  telemetry:
    metrics:
      level: none
```

> This configuration will be modified by Bindplane and should not be edited after the initial deployment.

4. Paste this into your `logging.yaml` file in the storage directory:

```yaml
output: stdout
level: info
```

5. Start the BDOT Collector using Docker Compose:

```bash
docker compose up -d
```

## Verifying the Installation

To verify that the collector is running correctly:

1. Check the container status:
```bash
docker compose ps
```

2. View the logs:
```bash
docker compose logs -f
```

## Configuring the BDOT Collector

Roll out a configuration update from Bindplane.

## Uninstalling

Stop Docker Compose and remove the BDOT Collector container.

```
docker compose down -v
docker compose rm -f bdot-collector
```
