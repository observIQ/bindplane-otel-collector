# Filelog receiver with Elasticsearch using JSON

This is example configuration shows the `filelog` receiver parsing JSON logs and sending those to an Elasticsearch cluster.

## Prerequisites

An Elasticsearch cluster/endpoint to send log data to.

## Configuration

An example configuration is located [here](./config.yaml).

1. Copy [config.yaml](./config.yaml) to `/opt/bindplane-otel-collector/config.yaml`
2. Modify `config.yaml` to include the Elasticsearch Exporter with endpoints to send log data to.
3. Restart the agent: `sudo systemctl restart bindplane-otel-collector`
