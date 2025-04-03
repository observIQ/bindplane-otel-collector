# Microsoft Sentinel Exporter

This exporter allows you to export logs to Microsoft Sentinel via the Log Analytics Ingestion API. Logs are exported in [OpenTelemetry Protocol JSON format](https://github.com/open-telemetry/opentelemetry-proto).

## Minimum Agent Versions
- Introduced: v1.73.1

## Supported Pipelines
- Logs

## How It Works

The Microsoft Sentinel exporter converts OpenTelemetry logs to JSON format and sends them to Microsoft Sentinel using the Log Analytics Ingestion API. Each log record is transformed to include a `TimeGenerated` field and the original log data under `resourceLogs`.

## Configuration
| Field         | Type   | Default | Required | Description                                                |
|---------------|--------|---------|----------|------------------------------------------------------------|
| endpoint      | string |         | ✓        | Microsoft Sentinel DCR or DCE endpoint                     |
| client_id     | string |         | ✓        | Azure client ID for authentication                         |
| client_secret | string |         | ✓        | Azure client secret for authentication                     |
| tenant_id     | string |         | ✓        | Azure tenant ID for authentication                         |
| rule_id       | string |         | ✓        | Data Collection Rule (DCR) ID or immutableId              |
| stream_name   | string |         | ✓        | Name of the custom log table in Microsoft Sentinel         |

## Example Configurations

### Minimal Configuration

```yaml
exporters:
  microsoftsentinel:
    endpoint: "<your-sentinel-endpoint>"
    client_id: "<your-client-id>"
    client_secret: "<your-client-secret>"
    tenant_id: "<your-tenant-id>"
    rule_id: "<your-dcr-id>"
    stream_name: "<your-stream-name>"
```

This configuration shows the minimum required fields to export logs to Microsoft Sentinel. All fields are required for the exporter to function properly.

## Setup

Before configuring the exporter, you'll need to set up several components in the Azure portal:

### 1. Create an Azure AD Application (not needed if you already have one)

1. Navigate to Azure Active Directory > App registrations
2. Click "New registration"
3. Give your application a name
4. Select supported account types (usually "Single tenant")
5. Click "Register"
6. After creation, note down the following:
   - Application (client) ID
   - Directory (tenant) ID
7. Under "Certificates & secrets":
   - Create a new client secret
   - Copy the secret value immediately (you won't be able to see it again)

### 2. Create a Log Analytics Workspace Table (not needed if you already have one setup)

1. Go to your Log Analytics workspace
2. Navigate to "Tables" under Settings
3. Click "New Custom Table"
4. Configure your table:
   - Give it a name (this will be your `stream_name`)
   - Select "JSON" as the data format
   - You will need to provide the following OTLP log formatted schema as an example
   ```json
   [
  {
    "TimeGenerated": "2023-01-02T03:04:05Z",
    "resourceLogs": [
      {
        "resource": {
          "attributes": [
            {
              "key": "service.name",
              "value": {
                "stringValue": "test-service"
              }
            },
            {
              "key": "host.name",
              "value": {
                "stringValue": "test-host"
              }
            }
          ]
        },
        "scopeLogs": [
          {
            "logRecords": [
              {
                "body": {
                  "stringValue": "Test log message"
                },
                "severityNumber": 9,
                "severityText": "INFO",
                "spanId": "",
                "timeUnixNano": "1672628645000000000",
                "traceId": ""
              }
            ],
            "scope": {
              "name": "test-scope",
              "version": "v1.0.0"
            }
          }
        ]
      }
    ]
  }
]
   ```
5. Click "Create"

### 3. Create a Data Collection Rule (DCR)

1. Navigate to Microsoft Sentinel
2. Go to Settings > Data Collection Rules
3. Click "Create"
4. Configure the DCR:
   - Select your subscription and resource group
   - Choose your Log Analytics workspace
   - Select the custom table you created
   - Set up any necessary transformations
5. After creation, note down:
   - The data collection rule (DCR) Endpoint URL (will be your `endpoint`), if you do not see an endpoint URL, try updating the API version on the json view of your DCR ruleset. If you still dont see it, you will need to set up a data collection endpoint (DCE). Additional information can be found here (https://learn.microsoft.com/en-us/azure/azure-monitor/logs/logs-ingestion-api-overview#data-collection-rule-dcr)
   - The Rule ID (will be your `rule_id`)

### 4. Set up Permissions

1. Go to your DCR
2. Navigate to "Access control (IAM)"
3. Add a role assignment:
   - Role: "Monitoring Metrics Publisher"
   - Assign access to: User, group, or service principal
   - Select your previously created Azure AD application (you may need to use the search functionality to find it)
4. Repeat the same for the Log Analytics workspace resource if needed.

Now you have all the required information to configure the exporter:
- `endpoint`: The DCR Endpoint URL
- `client_id`: The Application (client) ID
- `client_secret`: The secret value you created
- `tenant_id`: The Directory (tenant) ID
- `rule_id`: The DCR Rule ID
- `stream_name`: The name of your custom table

