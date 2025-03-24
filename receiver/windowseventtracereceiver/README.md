# Windows Event Trace Receiver

This receiver collects Event Tracing for Windows (ETW) events from Windows systems. It creates an ETW session and enables specified providers to capture real-time events.

## Supported Pipelines
- Logs

## How It Works
1. The receiver creates a session with the specified `session_name`.
2. It enables the specified providers for this session.
3. It listens for events from these providers in real-time.
4. Events are parsed and converted to Otel log records.
5. Logs get sent down the pipeline

## Prerequisites

- Enabling analytic/debug logs in the ETL format
- Knowledge of the ETW provider GUIDs to monitor

## Configuration
| Field         | Type     | Default           | Required | Description                                                 |
|---------------|----------|-------------------|----------|-------------------------------------------------------------|
| session_name  | string   | `OtelCollectorETW`| `false`  | The name to use for the ETW session.                        |
| providers     | []Provider | See below        | `false`  | A list of providers to subscribe to for ETW events.         |

### Provider Configuration
| Field | Type   | Default | Required | Description                                 |
|-------|--------|---------|----------|---------------------------------------------|
| name  | string |         | `true`   | The name or GUID of the ETW provider.       |

### Default Configuration
```yaml
receivers:
  windowseventtrace:
    session_name: OtelCollectorETW
    providers:
      # Microsoft-Windows-Kernel-File
      - name: "{EDD08927-9CC4-4E65-B970-C2560FB5C289}"
```

### Example Configuration
```yaml
receivers:
  windowseventtrace:
    session_name: CustomETWSession
    providers:
      # Microsoft-Windows-PowerShell
      - name: "{A0C1853B-5C40-4B15-8766-3CF1C58F985A}"
      # Microsoft-Windows-Security-Auditing
      - name: "{54849625-5478-4994-A5BA-3E3B0328C30D}"
exporters:
  googlecloud:
    project: my-gcp-project

service:
  pipelines:
    logs:
      receivers: [windowseventtrace]
      exporters: [googlecloud]
```

## Common ETW Providers
Here are some commonly used ETW providers:

| Provider Name | GUID |
|---------------|------|
| Microsoft-Windows-Kernel-File | {EDD08927-9CC4-4E65-B970-C2560FB5C289} |
| Microsoft-Windows-PowerShell | {A0C1853B-5C40-4B15-8766-3CF1C58F985A} |
| Microsoft-Windows-Security-Auditing | {54849625-5478-4994-A5BA-3E3B0328C30D} |
| Microsoft-Windows-DNS-Client | {1C95126E-7EEA-49A9-A3FE-A378B03DDB4D} |

You can find more providers on your hosts by running `logman query providers` in a PowerShell window with administrative privileges.
