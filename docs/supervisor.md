# Supervisor

The [OpenTelemetry supervisor](https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/cmd/opampsupervisor) is the process that runs the [OpenTelemetry collector](https://github.com/open-telemetry/opentelemetry-collector). The supervisor's responsibilities include but are not limited to:
- Starting & stopping the collector
- Communicating to OpAMP server on behalf of the collector
- Managing the collector's config based on OpAMP messages from the OpAMP server.
- Restarting the collector if it crashes

In the case of the BindPlane Agent, a custom OTel collector built using the [OpenTelemetry builder](https://github.com/open-telemetry/opentelemetry-collector/tree/main/cmd/builder) is used, the manifest for which can be found [here](../manifests/observIQ/README.md).

The install scripts are oriented towards running the supervisor connected to an OpAMP management platform, specifically [BindPlane](https://observiq.com/). The supervisor acts as a middle man between BindPlane and the collector and manages the collector's config.

## Configuration

The supervisor's config file can be located depending on your OS:

| OS      | Default Location                                                  |
|:--------|:------------------------------------------------------------------|
| Linux   | /opt/observiq-otel-collector/supervisor.yaml                      |
| Windows | C:\Program Files\observIQ OpenTelemetry Collector\supervisor.yaml |
| macOS   | /opt/observiq-otel-collector/supervisor.yaml                      |

Configuration options for the supervisor can be found [here](https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/main/cmd/opampsupervisor/specification/README.md#supervisor-configuration).

## Alternatives

If this model of running the collector via the supervisor and an OpAMP management platform doesn't work for your use case, you can opt to run the collector manually instead.

The collector's binary can be found depending on your OS below:

| OS      | Default Location                                                              |
|:--------|:------------------------------------------------------------------------------|
| Linux   | /opt/observiq-otel-collector/observiq-otel-collector                          |
| Windows | C:\Program Files\observIQ OpenTelemetry Collector\observiq-otel-collector.exe |
| macOS   | /opt/observiq-otel-collector/observiq-otel-collector                          |

You can create an OTel configuration for the collector and run it like any other OTel collector. For more information on OTel configurations, see the [OpenTelemetry docs](https://opentelemetry.io/docs/collector/configuration/).
