# Windows Installation

## Installing

To install the agent on Windows, run the PowerShell command below. The script automatically detects the system architecture (amd64 or arm64) and downloads the appropriate MSI.

> **Note:** The install script is available as of release v1.94.0. For earlier versions, see the [manual installation](#manual-installation) instructions below.

```pwsh
& ([scriptblock]::Create((Invoke-WebRequest -Uri "https://bdot.bindplane.com/latest/install_windows.ps1" -UseBasicParsing).Content))
```

To install a specific version, pass the `-Version` parameter:

```pwsh
& ([scriptblock]::Create((Invoke-WebRequest -Uri "https://bdot.bindplane.com/latest/install_windows.ps1" -UseBasicParsing).Content)) -Version "1.94.0"
```

For an interactive installation with the installer UI, add `-Interactive`:

```pwsh
& ([scriptblock]::Create((Invoke-WebRequest -Uri "https://bdot.bindplane.com/latest/install_windows.ps1" -UseBasicParsing).Content)) -Interactive
```

### Manual Installation

For versions prior to v1.94.0, or if you prefer to install without the script, download the MSI directly from `https://bdot.bindplane.com/v<version>/observiq-otel-collector.msi` (or `observiq-otel-collector-arm64.msi` for ARM64) and double click it to open the installation wizard.

Installation artifacts are signed. Information on verifying the signature can be found at [Verifying Artifact Signatures](./verify-signature.md).

### Managed Mode

To install the agent with an OpAMP connection configuration, pass the management flags to the install script:

```pwsh
& ([scriptblock]::Create((Invoke-WebRequest -Uri "https://bdot.bindplane.com/latest/install_windows.ps1" -UseBasicParsing).Content)) `
    -EnableManagement "1" `
    -OpAMPEndpoint "<your_endpoint>" `
    -OpAMPSecretKey "<secret-key>"
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

Locate the "observIQ Distro for OpenTelemetry Collector" service, right click the entry, and click "Restart" to restart the agent.

![The services dialog](./screenshots/windows/stop-restart-service.png)

Alternatively, the Powershell command below may be run to restart the agent service.
```pwsh
Restart-Service -Name "observiq-otel-collector"
```

## Stopping the Agent

Stopping the agent may be done through the services dialog.
To access the services dialog, press Win + R, enter `services.msc` into the Run dialog, and press enter.

![The run dialog](./screenshots/windows/launch-services.png)

Locate the "observIQ Distro for OpenTelemetry Collector" service, right click the entry, and click "Stop" to stop the agent.

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

To uninstall the agent, run the install script with the `-Uninstall` flag:

```pwsh
& ([scriptblock]::Create((Invoke-WebRequest -Uri "https://bdot.bindplane.com/latest/install_windows.ps1" -UseBasicParsing).Content)) -Uninstall
```

Alternatively, uninstall through the control panel via the "Uninstall a program" dialog.

![The control panel](./screenshots/windows/control-panel-uninstall.png)

Locate the `"observIQ Distro for OpenTelemetry Collector"` entry, and select uninstall.

![The uninstall or change a program dialog](./screenshots/windows/uninstall-collector.png)

Follow the wizard to complete removal of the agent.
