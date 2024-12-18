# Splunk Search API Receiver
This receiver collects Splunk events using the [Splunk Search API](https://docs.splunk.com/Documentation/Splunk/9.3.1/RESTREF/RESTsearch).

## Supported Pipelines
- Logs

## Prerequisites
- Splunk admin credentials
- Configured storage extension

## Use Case
Unlike other receivers, the SSAPI receiver is not built to collect live data. Instead, it collects a finite set of historical data and transfers it to a destination, preserving the timestamp from the source. For this reason, the SSAPI recevier only needs to be left running until all Splunk events have been migrated, which is denoted by the log message: "all search results exported". Until this log message or some other error is printed, avoid cancelling the collector for any reason, as it will unnecessarily interfere with the receiver's ability to protect against writing duplicate events.

## Configuration
| Field               | Type     | Default                                                                                         | Description                                                                                                                                                             |
|---------------------|----------|-------------------------------------------------------------------------------------------------|-------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| endpoint                   | string   | `required` `(no default)`                                                                 | The endpoint of the splunk instance to collect from.                                                                                                                                         |
| splunk_username            | string   | `(no default)`                                                                           | Specifies the username used to authenticate to Splunk using basic auth.                                                                                                           |
| splunk_password            | string   | `(no default)`                                                                           | Specifies the password used to authenticate to Splunk using basic auth.                                                                                                           |
| auth_token                    | string   | `(no default)`                                                                           | Specifies the token used to authenticate to Splunk using token auth. Mutually exclusive with basic auth using `splunk_username` and `splunk_password`. |
| token_type                    | string   | `(no default)`                                                                           | Specifies the type of token used to authenticate to Splunk using `auth_token`. Accepted values are "Bearer" or "Splunk". |
| job_poll_interval | duration | `5s` | The receiver uses an API call to determine if a search has completed. Specifies how long to wait between polling for search job completion. |
| searches.query | string | `required (no default)` | The Splunk search to run to retrieve the desired events. Queries must start with `search` and should not contain additional commands, nor any time fields (e.g. `earliesttime`) |
| searches.earliest_time | string | `required (no default)` | The earliest timestamp to collect logs. Only logs that occurred at or after this timestamp will be collected. Must be in ISO 8601 or RFC3339 format. |
| searches.latest_time | string | `required (no default)` | The latest timestamp to collect logs. Only logs that occurred at or before this timestamp will be collected. Must be in ISO 8601 or RFC3339 format. |
| searches.event_batch_size | int | `100` | The amount of events to query from Splunk for a single request. |
| storage | component | `required (no default)` | The component ID of a storage extension which can be used when polling for `logs`. The storage extension prevents duplication of data after an exporter error by remembering which events were previously exported. |

### Example Configuration
```yaml
receivers:
  splunksearchapi:
    endpoint: "https://splunk-c4-0.example.localnet:8089"
    tls:
      insecure_skip_verify: true
    splunk_username: "user"
    splunk_password: "pass"
    job_poll_interval: 5s
    searches:
      - query: 'search index=my_index'
        earliest_time: "2024-11-01T01:00:00.000-05:00"
        latest_time: "2024-11-30T23:59:59.999-05:00"
        event_batch_size: 500
    storage: file_storage

extensions:
  file_storage:
    directory: "./local/storage"
```

## How To

### Migrate historical events to Google Cloud Logging
1. Identify the Splunk index to migrate events from. Create a Splunk search to capture the events from that index. This will be the `searches.query` you pass to the receiver.
   - Example: `search index=my_index`
   - Note: queries must begin with the explicit `search` command, and must not include additional commands, nor any time fields (e.g. `earliesttime`)
2. Determine the timeframe you want to migrate events from, and set the `searches.earliest_time` and `searches.latest_time` config fields accordingly.
   - To migrate events from December 2024, EST (UTC-5):
     - `earliest_time: "2024-12-01T00:00:00.000-05:00"`
     - `latest_time: "2024-12-31T23:59:59.999-05:00"`
   - Note: By default, GCL will not accept logs with a timestamp older than 30 days. Contact Google to modify this rule.
3. Repeat steps 1 & 2 for each index you wish to collect from
3. Configure a storage extension to store checkpointing data for the receiver.
4. Configure the rest of the receiver fields according to your Splunk environment.
5. Add a `googlecloud` exporter to your config. Configure the exporter to send to a GCP project where your service account has Logging Admin role. To check the permissions of service accounts in your project, go to the [IAM page](https://console.cloud.google.com/iam-admin/iam). 
6. Disable the `sending_queue` field on the GCP exporter. The sending queue introduces an asynchronous step to the pipeline, which will jeopardize the receiver's ability to checkpoint correctly and recover from errors. For this same reason, avoid using any asynchronous processors (e.g., batch processor).

After following these steps, your configuration should look something like this:
```yaml
receivers:
  splunksearchapi:
    endpoint: "https://splunk-c4-0.example.localnet:8089"
    tls:
      insecure_skip_verify: true
    splunk_username: "user"
    splunk_password: "pass"
    job_poll_interval: 5s
    searches:
      - query: 'search index=my_index'
        earliest_time: "2024-12-01T00:00:00.000-05:00"
        latest_time: "2024-12-31T23:59:59.999-05:00"
        event_batch_size: 500
    storage: file_storage
exporters:
  googlecloud:
    project: "my-gcp-project"
    log: 
      default_log_name: "splunk-events"
    sending_queue:
      enabled: false

extensions:
  file_storage:
    directory: "./local/storage"

service:
  extensions: [file_storage]
  pipelines:
    logs:
      receivers: [splunksearchapi]
      exporters: [googlecloud]
```
You are now ready to migrate events from Splunk to Google Cloud Logging.