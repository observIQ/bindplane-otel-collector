# Mongodb Atlas Metrics with Google Cloud

The Mongodb Atlas Receiver can be used to send Mongodb Atlas metrics to Google Cloud Monitoring.

## Prerequisites

See the [documentation](https://github.com/observIQ/bindplane-otel-collector/blob/main/docs/receivers.md) for Mongodb Atlas prerequisites.

See the [prerequisites](../README.md) doc for Google Cloud prerequisites.

## Authentication Environment Variables

The configuration assumes the following environment variables are set:

- `MONGODB_ATLAS_PUBLIC_KEY`
- `MONGODB_ATLAS_PRIVATE_KEY`

Set the variables by creating a [systemd override](https://wiki.archlinux.org/title/systemd#Replacement_unit_files).

Run the following command

```bash
sudo systemctl edit bindplane-otel-collector
```

If this is the first time an override is being created, the file will be empty. Paste the following contents into the file. If the `Service` section is already present, append the two `Environment` lines to the `Service` section.

Replace `otel` with your Mongodb Atlas public key and private key.

```
[Service]
Environment=MONGODB_ATLAS_PUBLIC_KEY=otel
Environment=MONGODB_ATLAS_PRIVATE_KEY=otel
```

After restarting the agent, the configuration will attempt to use the configured public and private key.

```bash
sudo systemctl restart bindplane-otel-collector
```

## Warning

The Google Cloud Exporter appears to have trouble when datapoints for a metric are very close together (sometimes they are < 30s apart for MongoDB Atlas).
Because of this there will often be "Duplicate TimeSeries" errors logged, but generally the data should be ok.
