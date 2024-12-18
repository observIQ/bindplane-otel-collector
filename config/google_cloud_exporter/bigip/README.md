# Big IP with Google Cloud

The bigip Receiver can be used to send metrics to Google Cloud Monitoring.

## Prerequisites

See the [prerequisites](../README.md) doc for Google Cloud prerequisites.

## Limitations

To avoid duplicate metrics, ensure each agent is scraping a single BIG-IP system. This ensures
there is a unique `node_id` (hostname of the agent) label for each BIG-IP system being monitored.

## Configuration

An example configuration is located [here](./config.yaml).

1. Copy [config.yaml](./config.yaml) to `/opt/bindplane-otel-collector/config.yaml`.
2. Update the `endpoint` field with the endpoint of your Big IP F5 iControl REST API.
3. Follow the [authentication section](./README.md#authentication-environment-variables) for configuring username and password.
4. Restart the agent: `sudo systemctl restart bindplane-otel-collector`.

## Authentication Environment Variables

The configuration assumes the following environment variables are set:

- `BIGIP_USERNAME`
- `BIGIP_PASSWORD`

Set the variables by creating a [systemd override](https://wiki.archlinux.org/title/systemd#Replacement_unit_files).

Run the following command

```bash
sudo systemctl edit bindplane-otel-collector
```

If this is the first time an override is being created, the file will be empty. Paste the following contents into the file. If the `Service` section is already present, append the two `Environment` lines to the `Service` section.

Replace `otel` with your Big IP username and password.

```
[Service]
Environment=BIGIP_USERNAME=otel
Environment=BIGIP_PASSWORD=otel
```

After restarting the agent, the configuration will attempt to use the username:password `otel:otel`.

```bash
sudo systemctl restart bindplane-otel-collector
```
