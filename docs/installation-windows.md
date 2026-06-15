# Windows Installation

## Installing

To install the agent on Windows, run the PowerShell command below as an administrator. The script automatically detects the system architecture (amd64 or arm64) and downloads the appropriate MSI.

> **Note:** The install script is available as of release v2.0.1-beta.2. For earlier versions, see the [manual installation](#manual-installation) instructions below.

```pwsh
& ([scriptblock]::Create((New-Object System.Net.WebClient).DownloadString("https://bdot.bindplane.com/<version>/install_windows.ps1")))
```

To install a specific version, pass the `-Version` parameter:

```pwsh
& ([scriptblock]::Create((New-Object System.Net.WebClient).DownloadString("https://bdot.bindplane.com/<version>/install_windows.ps1"))) -Version "v1.96.0"
```

By default the script launches the installation wizard UI. To install silently (unattended), add `-Quiet`:

```pwsh
& ([scriptblock]::Create((New-Object System.Net.WebClient).DownloadString("https://bdot.bindplane.com/<version>/install_windows.ps1"))) -Quiet
```

### Manual Installation

For versions prior to v2.0.1-beta.2, or if you prefer to install without the script, download the MSI directly from `https://bdot.bindplane.com/v<version>/bindplane-otel-collector.msi` (or `bindplane-otel-collector-arm64.msi` for ARM64) and double click it to open the installation wizard.

Installation artifacts are signed, and the install script automatically verifies the MSI's Authenticode signature before installing. If verification fails, the script prompts before continuing (and aborts in `-Quiet` mode). To bypass verification, add `-SkipSignatureCheck` (not recommended). For more information on verifying the signature manually, see [Verifying Artifact Signatures](./verify-signature.md).

### OpAMP Management

To install the agent and connect the supervisor to an OpAMP management platform, set the following flags.

```pwsh
& ([scriptblock]::Create((New-Object System.Net.WebClient).DownloadString("https://bdot.bindplane.com/<version>/install_windows.ps1"))) `
    -EnableManagement "1" `
    -OpAMPEndpoint "<your_endpoint>" `
    -OpAMPSecretKey "<secret-key>"
```

To read more about OpAMP management, see the [supervisor docs](./supervisor.md).

### Script Parameters

The install script accepts the following parameters:

| Parameter | Description |
| --- | --- |
| `-Version <version>` | Version to install (e.g. `v2.0.1`). Omit or pass `latest` to install the newest release. |
| `-EnableManagement "1"` | Enable managed mode via OpAMP. Default is `0`. |
| `-OpAMPEndpoint <url>` | OpAMP server endpoint (e.g. `wss://app.bindplane.com/v1/opamp`). |
| `-OpAMPSecretKey <key>` | Secret key for OpAMP authentication. |
| `-OpAMPLabels <labels>` | Comma-separated `key=value` labels (e.g. `configuration=windows,env=prod`). |
| `-InstallDir <path>` | Custom installation directory. Defaults to the MSI default. |
| `-Clean "1"` | Remove existing configuration on install. Default is `0`. |
| `-Quiet` | Install silently without showing the installer UI. |
| `-SkipSignatureCheck` | Skip MSI Authenticode signature verification (not recommended). |
| `-Uninstall` | Uninstall the agent instead of installing it. |
| `-MsiUrl <url>` | Override the full MSI download URL. When set, `-Version` and architecture detection are ignored. |
| `-MsiFile <path>` | Install from a local MSI file. Skips all download and version resolution steps. |

The script must be run from an elevated (Administrator) PowerShell prompt.

## Configuring the Agent

After installing, the `bindplane-otel-collector` service will be running and ready for configuration!

The agent is ran and managed by the [OpenTelemetry supervisor](https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/cmd/opampsupervisor). The supervisor must receive the agent's configuration from an OpAMP management platform, after which it will stop and restart the agent with the new config.

The supervisor remembers the last config it received via OpAMP and always starts rewrites the agent's config file with it when it starts. This means you can't manually edit the agent's config file on disk. The best way to modify the configuration is to send a new one from the OpAMP platform the supervisor is connected to.

The agent configuration file is located at `C:\Program Files\observIQ OpenTelemetry Collector\supervisor_storage\effective.yaml`.

If this method of collector management does not work for your use case, see this [alternative option](./supervisor.md#alternatives)

**Logging**

Logs from the agent will appear in `<install_dir>/supervisor_storage/agent.log` (`C:\Program Files\observIQ OpenTelemetry Collector\supervisor_storage\agent.log` by default).

Stderr for the supervisor process can be found at `<install_dir>/log/observiq_collector.err` (`C:\Program Files\observIQ OpenTelemetry Collector\log\observiq_collector.err` by default).

## Restarting the Agent

Restarting the agent may be done through the services dialog.
To access the services dialog, press Win + R, enter `services.msc` into the Run dialog, and press enter.

![The run dialog](./screenshots/windows/launch-services.png)

Locate the "BDOT" service, right click the entry, and click "Restart" to restart the agent.

![The services dialog](./screenshots/windows/stop-restart-service.png)

Alternatively, the Powershell command below may be run to restart the agent service.

```pwsh
Restart-Service -Name "bindplane-otel-collector"
```

## Stopping the Agent

Stopping the agent may be done through the services dialog.
To access the services dialog, press Win + R, enter `services.msc` into the Run dialog, and press enter.

![The run dialog](./screenshots/windows/launch-services.png)

Locate the "BDOT" service, right click the entry, and click "Stop" to stop the agent.

![The services dialog](./screenshots/windows/stop-restart-service.png)

Alternatively, the Powershell command below may be run to stop the agent service.

```pwsh
Stop-Service -Name "bindplane-otel-collector"
```

## Starting the Agent

Starting the agent may be done through the services dialog.
To access the services dialog, press Win + R, enter `services.msc` into the Run dialog, and press enter.

![The run dialog](./screenshots/windows/launch-services.png)

Locate the "BDOT" service, right click the entry, and click "Start" to start the agent.

![The services dialog](./screenshots/windows/start-service.png)

Alternatively, the Powershell command below may be run to start the agent service.

```pwsh
Start-Service -Name "bindplane-otel-collector"
```

## Uninstalling

To uninstall the agent, run the install script with the `-Uninstall` flag:

```pwsh
& ([scriptblock]::Create((New-Object System.Net.WebClient).DownloadString("https://bdot.bindplane.com/<version>/install_windows.ps1"))) -Uninstall
```

Alternatively, uninstall through the control panel via the "Uninstall a program" dialog.

![The control panel](./screenshots/windows/control-panel-uninstall.png)

Locate the `"BindPlane Distro for OpenTelemetry Collector (BDOT)"` entry, and select uninstall.

![The uninstall or change a program dialog](./screenshots/windows/uninstall-collector.png)

Follow the wizard to complete removal of the agent.

Alternatively, the Powershell command below may be run to uninstall the agent. It resolves the installed product via the collector's stable upgrade code (defined in `windows/wix.json`), which keeps working across product renames, and then removes it with `msiexec`.

```pwsh
$installer = New-Object -ComObject WindowsInstaller.Installer
$productCode = $installer.RelatedProducts("{D67CCA1A-6708-4096-8BDE-5069739FB861}") | Select-Object -First 1
Start-Process msiexec.exe -ArgumentList "/x", $productCode -Wait
```
