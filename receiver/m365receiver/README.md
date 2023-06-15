# Microsoft Office 365 Receiver
Receives metrics from [Microsoft Office 365](https://www.microsoft365.com/)
via the [Microsoft Graph API](https://learn.microsoft.com/en-us/graph/api/overview?view=graph-rest-1.0&preserve-view=true),
and logs via the [Microsoft Management API](https://learn.microsoft.com/en-us/office/office-365-management-api/office-365-management-activity-api-schema).

## Minimum Collector Versions
- Introduced: [v1.27.0](https://github.com/observIQ/observiq-otel-collector/releases/tag/v1.27.0)

## Supported Pipelines
- Metrics
- Logs

## How It Works
1. The user configures their instance of Microsoft Office to enable monitoring of metrics, logs, or both.
2. The user configures this receiver in a pipeline.
3. The user configures a supported component to route telemetry from this receiver.

## Prerequisites
- Created instance of Microsoft Office 365 with the following subscriptions: Microsoft 365 Business Basic, Microsoft 365 E5 Compliance, Microsoft 365 E3 (Works with the respective "upgraded" versions as well.)
- Access to an Admin account for the instance of 365 to be monitored.

## Configuration
| Field               | Type     | Default                                                                                  | Description                                                                                                                                                             |
|---------------------|----------|------------------------------------------------------------------------------------------|-------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| tenant_id | string | `(no default)` | `required` Identifies the instance of 365 to be monitored by this receiver. Needed for metrics and logs. |
| client_id | string | `(no default)` | `required` The identifier this receiver will use to monitor the given tenant/instance. Needed for metrics and logs. |
| client_secret | string | `(no default)` | `required` The private key this receiver will use, must belong to the given client_id. Needed for metrics and logs. |
| logs | object | `(n/a)` | Configuration object for other fields listed below. |
| logs.poll_interval | duration | `5m` | The receiver collects logs on an interval. Value must be in minutes (i.e. `10m`, `120m`). Can be omitted for default interval of 5 minutes. |
| logs.general | bool | `true` | Indicates whether or not logs should be collected from the General audit/content blob. Can be omitted to indicate true. | 
| logs.exchange | bool | `true` | Indicates whether or not logs should be collected from the Exchange audit/content blob. Can be omitted to indicate true. |    
| logs.sharepoint | bool | `true` | Indicates whether or not logs should be collected from the SharePoint audit/content blob. Can be omitted to indicate true. |  
| logs.azureAD | bool | `true` | Indicates whether or not logs should be collected from the Azure Active Directory audit/content blob. Can be omitted to indicate true. | 
| logs.dlp | bool | `true` | Indicates whether or not logs should be collected from the Data Loss Prevention audit/content blob. Can be omitted to indicate true. | 
| storage | component | `(no default)` | The component ID of a storage extension which can be used when polling for `logs` . The storage extension prevents duplication of data after a collector restart by remembering which data were previously collected. No storage is used when omitted.                         

## Example Configurations

### Collect metrics: 
```yaml
receivers:
  m365:
    tenant_id: tenant_id
    client_id: client_id
    client_secret: client_secret
exporters:
  file/no_rotation:
    path: /some/file/path/foo.json
service:
  pipelines:
    metrics:
      receivers: [m365]
      exporters: [file/no_rotation]
```

### Collect logs (default values):
```yaml
receivers:
  m365:
    tenant_id: tenant_id
    client_id: client_id
    client_secret: client_secret
exporters:
  file/no_rotation:
    path: /some/file/path/foo.json
service:
  pipelines:
    logs:
      receivers: [m365]
      exporters: [file/no_rotation]
```

### Collect logs (custom poll interval, storage component, only sharepoint & azureAD logs):
```yaml
receivers:
  m365:
    tenant_id: tenant_id
    client_id: client_id
    client_secret: client_secret
    logs:
      poll_interval: 10m
      general: false
      exchange: false
      dlp: false
    storage: file_storage
exporters:
  file/no_rotation:
    path: /some/file/path/foo.json
service:
  pipelines:
    logs:
      receivers: [m365]
      exporters: [file/no_rotation]
```

## How To
### Configuring Office 365
The steps below outline how to configure Office 365 to allow the receiver to collect metrics from it. 
To use this receiver, the instance of Office 365 needs the following subscriptions: **Microsoft 365 Business Basic**, **Microsoft 365 E5 Compliance**, and **Microsoft 365 E3**. (Works with the respective "upgraded" versions as well.)

1. **Login to Azure:** Log in to Microsoft Azure under an Admin account for the instance of 365 to be monitored.
2. **Register the receiver in Azure AD:** Navigate to Azure Active Directory. Then go to "App Registrations" and select "New Registration". 
Give the app a descriptive name like "365 Receiver". For "Supported account types", select the Single Tenant option and leave the Redirect URL empty.
3. **Add API Permissions:** Select "View API Permissions" beneath the general application info and click "Add Permissions". The permissions needed for metrics and logs differ, so for whichever monitoring is needed the respective permissions are outlined below.
    - **Metrics:** Select "Microsoft Graph", then "Application Permissions". Find the "Reports" tab and select "Reports.Read.All". Click "Add Permissions" at the bottom of the panel.
    - **Logs:** Select "Office 365 Management APIs", then "Application Permissions". Now select the "ActivityFeed.Read", "ActivityFeed.ReadDlp", and "ServiceHealth.Read" permissions. Click "Add Permissions" at the bottom of the panel.
4. **Grant Admin Consent:** Select the "Grant admin consent for {organization}" button and confirm the pop-up. This will allow the application to access the data returned by the Microsoft Graph and Office 365 Management APIs.
5. **Generate Client Secret:** Select the "Certificates & secrets" tab in the left panel. Under the "Client Secrets" tab, select "New Client Secret." Give it a meaningful description and select the recommended period of 180 days. Save the text in the "Value" column since this is the only time that value will be accessible.
    - **Note:** The receiver will need to be reconfigured with a newly generated Client Secret once the initial one expires.
6. **Save Client ID and Tenant ID values:** You will also need the "client_id" value found on the information page for the application that was created. The value will be listed as "Application (client) id." You will also need the tenant value which will be listed as "Directory (tenant) id." Save these values for later.

**Note: The first time an instance of Microsoft Office 365 is set up for monitoring, an extra step for collecting logs is required.**
1. Log into [Microsoft Purview Compliance Portal](https://compliance.microsoft.com) with an admin account.
2. Navigate to "Solutions" then ["Audit"](https://compliance.microsoft.com/auditlogsearch).
3. If auditing is not turned on for your organization, a banner is displayed prompting you start recording user and admin activity.
4. Select "Start recording user and admin activity".
**It will take up to 60 minutes for the change to take effect, so until that point do not run the receiver with logs turned on or else it will fail.**

After following the above steps, the instance of Microsoft Office 365 is ready for monitoring and the receiver can now be configured.

## Notes
- The metrics scraper only runs once every 24 hours because of the nature of the data returned by Microsoft Office
