# GCS Pub/Sub Event Receiver

The GCS Pub/Sub Event Receiver consumes GCS event notifications for object finalization events (`OBJECT_FINALIZE`) delivered via Google Cloud Pub/Sub and emits the GCS object's contents as log records.

## How It Works

1. The receiver subscribes to a Pub/Sub subscription that is configured to receive GCS event notifications.
2. When an `OBJECT_FINALIZE` event is received (equivalent to a new object being created or overwritten), the receiver downloads the GCS object.
3. The receiver parses the object's contents into log records, using JSON, Avro OCF, or line-based parsing depending on the file type.
4. All other event types (e.g., `OBJECT_DELETE`, `OBJECT_METADATA_UPDATE`) are acknowledged and ignored.
5. If the GCS object does not exist or cannot be accessed due to a permission error, the Pub/Sub message is nacked to trigger Dead Letter Queue (DLQ) processing.

## Ack Deadline Extension Behavior

The Pub/Sub client library automatically manages ack deadline extensions while a message is being processed. The `max_extension` configuration controls how long the client will keep extending the deadline before giving up:

1. **Automatic Extension**: The Pub/Sub client library extends the ack deadline in the background as long as the message is being processed.
2. **Maximum Extension**: Extensions stop after the duration specified by `max_extension` (default: 1 hour). After this point, the message will be nacked and redelivered.
3. **Concurrency**: The `workers` setting controls `MaxOutstandingMessages` in the Pub/Sub receive settings, limiting how many messages are processed in parallel.

This approach ensures that:

- Messages remain invisible to other consumers during processing.
- Long-running downloads and parsing operations do not cause premature redelivery.
- Messages are eventually redelivered if processing takes too long.

## Dead Letter Queue (DLQ) Behavior

Certain error conditions cause the receiver to nack a message rather than ack it. When a Pub/Sub subscription has a Dead Letter Topic configured, repeated nacks will route the message there:

| Condition | Behavior |
|---|---|
| GCS object not found | Nack (DLQ) |
| IAM permission denied (HTTP 403) | Nack (DLQ) |
| Unsupported file type | Nack (DLQ) |
| Other errors (network, transient) | Nack (retry) |

## File Format Support

The receiver automatically detects the file format based on the file extension and content type:

| Format | Detection |
|---|---|
| Avro OCF | `.avro`, `.avro.gz` extension or `application/avro` content type |
| JSON | `.json`, `.json.gz` extension or `application/json` content type |
| Plain text | All other files; parsed line by line |

Gzip-compressed files are transparently decompressed before parsing.

## Configuration

| Field | Type | Default | Required | Description |
|---|---|---|---|---|
| `project_id` | string | | `true` | The Google Cloud project ID containing the Pub/Sub subscription |
| `subscription_id` | string | | `true` | The Pub/Sub subscription ID that receives GCS event notifications |
| `credentials_file` | string | | `false` | Path to a Google Cloud service account credentials JSON file. If empty, Application Default Credentials (ADC) are used |
| `workers` | int | `5` | `false` | The number of concurrent workers to process Pub/Sub messages in parallel |
| `max_extension` | duration | `1h` | `false` | The maximum duration for which the Pub/Sub client will extend the ack deadline for a message being processed |
| `max_log_size` | int | `1048576` | `false` | The maximum size in bytes for a single log record. Logs exceeding this size will be split into chunks |
| `max_logs_emitted` | int | `1000` | `false` | The maximum number of log records to emit in a single batch. Higher values reduce batches but increase memory usage |
| `storage` | component.ID | | `false` | The ID of a storage extension to use for tracking per-object byte offsets, enabling resumption of interrupted object reads |
| `bucket_name_filter` | string | | `false` | A Go regular expression to filter GCS events by bucket name. Only events from matching buckets are processed |
| `object_key_filter` | string | | `false` | A Go regular expression to filter GCS events by object name. Only objects with matching names are processed |

## GCP Setup

### Step 1: Enable GCS Pub/Sub Notifications

Configure your GCS bucket to publish object change notifications to a Pub/Sub topic using the `gcloud` CLI:

```bash
# Grant the GCS service account permission to publish to the topic
GCS_SERVICE_ACCOUNT=$(gsutil kms serviceaccount -p YOUR_PROJECT_ID)
gcloud pubsub topics add-iam-policy-binding YOUR_TOPIC_ID \
  --member="serviceAccount:${GCS_SERVICE_ACCOUNT}" \
  --role="roles/pubsub.publisher"

# Enable GCS notifications for OBJECT_FINALIZE events on the bucket
gcloud storage buckets notifications create gs://YOUR_BUCKET_NAME \
  --topic=YOUR_TOPIC_ID \
  --event-types=OBJECT_FINALIZE
```

### Step 2: Create a Pub/Sub Subscription

```bash
gcloud pubsub subscriptions create YOUR_SUBSCRIPTION_ID \
  --topic=YOUR_TOPIC_ID \
  --ack-deadline=60
```

For production use, consider also configuring a Dead Letter Topic on the subscription so that repeatedly failing messages are routed there instead of being retried indefinitely.

### Required IAM Permissions

The identity running the collector (service account or ADC principal) requires:

```
roles/pubsub.subscriber  (on the Pub/Sub subscription)
roles/storage.objectViewer  (on the GCS bucket)
```

Or the equivalent individual permissions:

| Permission | Resource | Purpose |
|---|---|---|
| `pubsub.subscriptions.consume` | Subscription | Receive and acknowledge Pub/Sub messages |
| `storage.objects.get` | Bucket / Objects | Download GCS objects |

### Authentication

The receiver supports two authentication methods:

1. **Application Default Credentials (ADC)** (recommended): Leave `credentials_file` empty. The collector will use the ambient credentials from the environment (Workload Identity, service account key file pointed to by `GOOGLE_APPLICATION_CREDENTIALS`, etc.).

2. **Explicit credentials file**: Set `credentials_file` to the path of a service account key JSON file.

## Example Configurations

### Minimal Configuration

```yaml
receivers:
  gcsevent:
    project_id: my-gcp-project
    subscription_id: my-gcs-events-sub

exporters:
  otlp:
    endpoint: otelcol:4317

service:
  pipelines:
    logs:
      receivers: [gcsevent]
      exporters: [otlp]
```

### Full Configuration with Filters and Storage

```yaml
receivers:
  gcsevent:
    project_id: my-gcp-project
    subscription_id: my-gcs-events-sub
    credentials_file: /etc/collector/gcp-credentials.json
    workers: 10
    max_extension: 2h
    max_log_size: 4194304      # 4 MB
    max_logs_emitted: 500
    storage: file_storage
    bucket_name_filter: "^prod-logs-"
    object_key_filter: "\.json(\.gz)?$"

extensions:
  file_storage:
    directory: /var/lib/otelcol/storage

exporters:
  otlp:
    endpoint: otelcol:4317

service:
  extensions: [file_storage]
  pipelines:
    logs:
      receivers: [gcsevent]
      exporters: [otlp]
```

## Internal Telemetry

This receiver emits the following internal metrics:

| Metric | Type | Description |
|---|---|---|
| `otelcol_gcsevent.batch_size` | Histogram | Number of log records in each emitted batch |
| `otelcol_gcsevent.objects_handled` | Sum | Total number of GCS objects successfully processed |
| `otelcol_gcsevent.failures` | Sum | Number of transient processing failures |
| `otelcol_gcsevent.parse_errors` | Sum | Number of individual log records skipped due to parse errors within a GCS object |
| `otelcol_gcsevent.dlq_file_not_found_errors` | Sum | Number of messages nacked due to the GCS object not being found |
| `otelcol_gcsevent.dlq_iam_errors` | Sum | Number of messages nacked due to IAM permission denied errors |
| `otelcol_gcsevent.dlq_unsupported_file_errors` | Sum | Number of messages nacked due to unsupported file types |
