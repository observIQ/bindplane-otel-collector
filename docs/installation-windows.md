# Windows Installation

## Installing

To install the agent on Windows, run the PowerShell command below from an **elevated (Administrator)** PowerShell prompt. The script automatically detects the system architecture (amd64 or arm64) and downloads the appropriate MSI. By default it shows the installer UI.

> **Note:** The install script is available as of release v1.96.0. For earlier versions, see the [manual installation](#manual-installation) instructions below.

```pwsh
& ([scriptblock]::Create((New-Object System.Net.WebClient).DownloadString("https://bdot.bindplane.com/<version>/install_windows.ps1")))
```

To install a specific version, pass the `-Version` parameter:

```pwsh
& ([scriptblock]::Create((New-Object System.Net.WebClient).DownloadString("https://bdot.bindplane.com/<version>/install_windows.ps1"))) -Version "v1.96.0"
```

For an unattended (silent) installation without the installer UI, add `-Quiet`:

```pwsh
& ([scriptblock]::Create((New-Object System.Net.WebClient).DownloadString("https://bdot.bindplane.com/<version>/install_windows.ps1"))) -Quiet
```

### Additional Options

The script accepts the following parameters:

| Parameter | Description |
| --- | --- |
| `-Version` | The version to install (e.g. `v1.96.0`). Omit or pass `latest` to install the latest release. |
| `-Quiet` | Run the installer silently instead of showing the installer UI. |
| `-InstallDir` | Custom installation directory. Defaults to the MSI default. |
| `-Clean` | Set to `"1"` to remove existing configuration on install. Default is `"0"`. |
| `-EnableManagement` | Set to `"1"` to enable managed mode via OpAMP. Default is `"0"`. See [Managed Mode](#managed-mode). |
| `-OpAMPEndpoint` | OpAMP server endpoint URL (e.g. `wss://app.bindplane.com/v1/opamp`). |
| `-OpAMPSecretKey` | Secret key for OpAMP authentication. |
| `-OpAMPLabels` | Comma-separated `key=value` labels for OpAMP (e.g. `configuration=windows,env=prod`). |
| `-MsiUrl` | Override the full MSI download URL. When set, `-Version` and architecture detection are ignored. |
| `-MsiFile` | Path to a local MSI file to install. Skips all download and version resolution steps. |
| `-SkipSignatureCheck` | Skip MSI Authenticode signature verification. |
| `-Uninstall` | Uninstall the agent instead of installing it. See [Uninstalling](#uninstalling). |

### Manual Installation

For versions prior to v1.96.0, or if you prefer to install without the script, download the MSI directly from `https://bdot.bindplane.com/v<version>/observiq-otel-collector.msi` (or `observiq-otel-collector-arm64.msi` for ARM64) and double click it to open the installation wizard.

### Signature Verification

Installation artifacts are signed, and the script verifies the MSI's Authenticode signature before installing. If verification fails during an interactive install, the script warns and prompts before continuing; during a `-Quiet` install it aborts. To bypass verification entirely, pass `-SkipSignatureCheck` (not recommended). More information on verifying signatures can be found at [Verifying Artifact Signatures](./verify-signature.md).

### Managed Mode

To install the agent with an OpAMP connection configuration, pass the management flags to the install script:

```pwsh
& ([scriptblock]::Create((New-Object System.Net.WebClient).DownloadString("https://bdot.bindplane.com/<version>/install_windows.ps1"))) `
    -EnableManagement "1" `
    -OpAMPEndpoint "<your_endpoint>" `
    -OpAMPSecretKey "<secret-key>"
```

To read more about the generated connection configuration file see [OpAMP docs](./opamp.md).

## Configuring the Agent

After installing, the `observiq-otel-collector` service will be running and ready for configuration!

The agent logs to `C:\Program Files\observIQ OpenTelemetry Collector\log\collector.log` by default.

By default, the config file for the agent can be found at `C:\Program Files\observIQ OpenTelemetry Collector\config.yaml`. When changing the configuration, the agent service must be restarted in order for config changes to take effect.

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

Alternatively, the PowerShell command below may be run to restart the agent service.
```pwsh
Restart-Service -Name "observiq-otel-collector"
```

## Stopping the Agent

Stopping the agent may be done through the services dialog.
To access the services dialog, press Win + R, enter `services.msc` into the Run dialog, and press enter.

![The run dialog](./screenshots/windows/launch-services.png)

Locate the "observIQ Distro for OpenTelemetry Collector" service, right click the entry, and click "Stop" to stop the agent.

![The services dialog](./screenshots/windows/stop-restart-service.png)

Alternatively, the PowerShell command below may be run to stop the agent service.
```pwsh
Stop-Service -Name "observiq-otel-collector"
```

## Starting the Agent

Starting the agent may be done through the services dialog.
To access the services dialog, press Win + R, enter `services.msc` into the Run dialog, and press enter.

![The run dialog](./screenshots/windows/launch-services.png)

Locate the "observIQ Distro for OpenTelemetry Collector" service, right click the entry, and click "Start" to start the agent.

![The services dialog](./screenshots/windows/start-service.png)

Alternatively, the PowerShell command below may be run to start the agent service.
```pwsh
Start-Service -Name "observiq-otel-collector"
```

## Uninstalling

To uninstall the agent, run the install script with the `-Uninstall` flag:

```pwsh
& ([scriptblock]::Create((New-Object System.Net.WebClient).DownloadString("https://bdot.bindplane.com/<version>/install_windows.ps1"))) -Uninstall
```

Alternatively, uninstall through the control panel via the "Uninstall a program" dialog.

![The control panel](./screenshots/windows/control-panel-uninstall.png)

Locate the `"observIQ Distro for OpenTelemetry Collector"` entry, and select uninstall.

![The uninstall or change a program dialog](./screenshots/windows/uninstall-collector.png)

Follow the wizard to complete removal of the agent.
