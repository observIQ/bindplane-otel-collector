# Azure Log Analytics Exporter

This exporter allows you to export logs to Azure Log Analytics via the Log Analytics Ingestion API. Logs are exported in [OpenTelemetry Protocol JSON format](https://github.com/open-telemetry/opentelemetry-proto) if the raw_log_field is not supplied, otherwise they are supplied in the form 
```json
[
  {
    "RawData": "<log data from field specified in raw_log_field>"
  }
]
 ```

## Minimum Agent Versions
- Introduced: v1.73.1

## Supported Pipelines
- Logs

## How It Works

This exporter sends logs to Azure Log Analytics using the [Log Analytics Ingestion API](https://learn.microsoft.com/en-us/azure/azure-monitor/logs/logs-ingestion-api-overview). Before using the exporter, you must configure a Data Collection Rule (DCR) or Data Collection Endpoint (DCE) and a custom table within your Log Analytics workspace.

The required schema for the custom table depends on the `raw_log_field` configuration option:

*   **Default (OTLP JSON Format):** If `raw_log_field` is *not* specified, the exporter sends logs in the standard [OpenTelemetry Protocol (OTLP) JSON format](https://github.com/open-telemetry/opentelemetry-proto). Your custom table must be configured with a schema compatible with this OTLP JSON structure (see the Setup section for an example).
*   **Raw Log Mode:** If `raw_log_field` *is* specified, the exporter extracts the data from the designated field and sends logs in the following simple JSON format:
    ```json
    [
      {
        "RawData": "<log data from field specified in raw_log_field>"
      }
    ]
    ```
    In this case, your custom table must have a column named `RawData` to store the log content.

  In both cases, a TimeGenerated field will automatically be added to the schema as it is required.

## Configuration
| Field         | Type   | Default | Required | Description                                                |
|---------------|--------|---------|----------|------------------------------------------------------------|
| endpoint      | string |         | ✓        | Azure Log Analytics DCR or DCE endpoint                     |
| client_id     | string |         | ✓        | Azure client ID for authentication                         |
| raw_log_field | string | ""        |         | Name of the log field to specifically send to log analytics|
| client_secret | string |         | ✓        | Azure client secret for authentication                     |
| tenant_id     | string |         | ✓        | Azure tenant ID for authentication                         |
| rule_id       | string |         | ✓        | Data Collection Rule (DCR) ID or immutableId              |
| stream_name   | string |         | ✓        | Name of the custom log table in Log Analytics       |

## Example Configurations
```yaml
exporters:
  azureloganalytics:
    endpoint: "<your-log-ingestion-endpoint>"
    client_id: "<your-client-id>"
    client_secret: "<your-client-secret>"
    tenant_id: "<your-tenant-id>"
    raw_log_field: body
    rule_id: "<your-dcr-id>"
    stream_name: "<your-stream-name>"
```
### Minimal Configuration

```yaml
exporters:
  azureloganalytics:
    endpoint: "<your-log-ingestion-endpoint>"
    client_id: "<your-client-id>"
    client_secret: "<your-client-secret>"
    tenant_id: "<your-tenant-id>"
    rule_id: "<your-dcr-id>"
    stream_name: "<your-stream-name>"
```

This configuration shows the minimum required fields to export logs to Azure Log Analytics. All fields are required for the exporter to function properly.

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
   - Give it a name (this will be the display name in the Azure portal). **Important:** The actual `stream_name` value used in the exporter configuration must be prefixed with `Custom-`. For example, if you name the table `my_logs` in the portal, the `stream_name` configuration value should be `Custom-my_logs`.
   - Select "JSON" as the data format
   - Provide an example schema based on your configuration:
     - **If `raw_log_field` is NOT set (Default):** Use the following OTLP log formatted schema:
       ```json
       [
         {
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
     - **If `raw_log_field` IS set:** Use the following simple schema with a `RawData` field:
       ```json
       [
         {
           "RawData": "Sample log entry content"
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

