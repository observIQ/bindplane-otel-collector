# Windows Installation

## Installing

To install the agent on Windows run the Powershell command below to install the MSI with no UI.

```pwsh
msiexec /i "https://github.com/observIQ/bindplane-otel-collector/releases/latest/download/observiq-otel-collector.msi" /quiet
```

Alternately, for an interactive installation [download the latest MSI](https://github.com/observIQ/bindplane-otel-collector/releases/latest).

After downloading the MSI, simply double click it to open the installation wizard. Follow the instructions to configure and install the agent.

Installation artifacts are signed. Information on verifying the signature can be found at [Verifying Artifact Signatures](./verify-signature.md).

### Managed Mode

To install the agent with an OpAMP connection configuration set the following flags.

```sh
msiexec /i "https://github.com/observIQ/bindplane-otel-collector/releases/latest/download/observiq-otel-collector.msi" /quiet ENABLEMANAGEMENT=1 OPAMPENDPOINT=<your_endpoint> OPAMPSECRETKEY=<secret-key>
```

To read more about the generated connection configuration file see [OpAMP docs](./opamp.md).

## Configuring the Agent

After installing, the `observiq-otel-collector` service will be running and ready for configuration!

The agent logs to `C:\Program Files\observIQ OpenTelemetry Collector\log\collector.log` by default.

By default, the config file for the agent can be found at `C:\Program Files\observIQ OpenTelemetry Collector\config.yaml`. When changing the configuration,the agent service must be restarted in order for config changes to take effect.

For more information on configuring the agent, see the [OpenTelemetry docs](https://opentelemetry.io/docs/collector/configuration/).

**Logging**

Logs from the agent will appear in `<install_dir>/log` (`C:\Program Files\observIQ OpenTelemetry Collector\log` by default).

Stderr for the agent process can be found at `<install_dir>/log/observiq_collector.err` (`C:\Program Files\observIQ OpenTelemetry Collector\log\observiq_collector.err` by default).

## Restarting the Agent

Restarting the agent may be done through the services dialog.
To access the services dialog, press Win + R, enter `services.msc` into the Run dialog, and press enter.

![The run dialog](./screenshots/windows/launch-services.png)

Locate the "Bindplane Distro for OpenTelemetry Collector" service, right click the entry, and click "Restart" to restart the agent.

![The services dialog](./screenshots/windows/stop-restart-service.png)

Alternatively, the Powershell command below may be run to restart the agent service.

```pwsh
Restart-Service -Name "observiq-otel-collector"
```

## Stopping the Agent

Stopping the agent may be done through the services dialog.
To access the services dialog, press Win + R, enter `services.msc` into the Run dialog, and press enter.

![The run dialog](./screenshots/windows/launch-services.png)

Locate the "Bindplane Distro for OpenTelemetry Collector" service, right click the entry, and click "Stop" to stop the agent.

![The services dialog](./screenshots/windows/stop-restart-service.png)

Alternatively, the Powershell command below may be run to stop the agent service.

```pwsh
Stop-Service -Name "observiq-otel-collector"
```

## Starting the Agent

Starting the agent may be done through the services dialog.
To access the services dialog, press Win + R, enter `services.msc` into the Run dialog, and press enter.

![The run dialog](./screenshots/windows/launch-services.png)

Locate the "observIQ Distro for OpenTelemetry Collector" service, right click the entry, and click "Start" to start the agent.

![The services dialog](./screenshots/windows/start-service.png)

Alternatively, the Powershell command below may be run to start the agent service.

```pwsh
Start-Service -Name "observiq-otel-collector"
```

## Uninstalling

To uninstall the agent on Windows, navigate to the control panel, then to the "Uninstall a program" dialog.

![The control panel](./screenshots/windows/control-panel-uninstall.png)

Locate the `"Bindplane Distro for OpenTelemetry Collector"` entry, and select uninstall.

![The uninstall or change a program dialog](./screenshots/windows/uninstall-collector.png)

Follow the wizard to complete removal of the agent.

Alternatively, Powershell command below may be run to uninstall the agent.

```pwsh
(Get-WmiObject -Class Win32_Product -Filter "Name = 'observIQ Distro for OpenTelemetry Collector'").Uninstall()
```
