# REST API Receiver

The REST API receiver is a generic receiver that can pull data from any REST API endpoint. It supports both logs and metrics collection, with configurable authentication, pagination, and time-based offset tracking.

## Supported Pipelines

Alpha:
- Logs
- Metrics

**Due to the wide range of use cases possible for this receiver, this component offers a best-effort integration with common API patterns, but may not completely align with every REST API.**

## How It Works

1. The receiver polls a configured REST API endpoint at a specified interval.
2. It handles authentication (None, API Key, Bearer Token, Basic Auth, OAuth2, or Akamai EdgeGrid).
3. It supports pagination to fetch all available data.
4. It can track time-based offsets to avoid duplicate data collection.
5. It converts JSON responses to OpenTelemetry logs or metrics.
6. It optionally uses storage extension for checkpointing to resume after restarts.

## Prerequisites

- A REST API endpoint that returns JSON data
- Appropriate authentication credentials (if required)
- Optional: Storage extension for checkpointing (recommended for production)

## Configuration

| Field | Type | Default | Required | Description |
|-------|------|---------|----------|-------------|
| `url` | string | | `true` | The base URL for the REST API endpoint |
| `response_field` | string | | `false` | The name of the field in the response that contains the array of items. If empty, the response is assumed to be a top-level array. For nested fields, use dot notation (e.g., `response.data`) |
| `metrics` | object | | `false` | Metrics configuration (see below) |
| `auth_mode` | string | `none` | `false` | Authentication mode: `none`, `apikey`, `bearer`, `basic`, `oauth2`, or `akamai_edgegrid` |
| `apikey` | object | | `false` | API Key configuration (see below) |
| `bearer` | object | | `false` | Bearer Token configuration (see below) |
| `basic` | object | | `false` | Basic Auth configuration (see below) |
| `oauth2` | object | | `false` | OAuth2 Client Credentials configuration (see below) |
| `akamai_edgegrid` | object | | `false` | Akamai EdgeGrid configuration (see below) |
| `pagination` | object | | `false` | Pagination configuration (see below) |
| `min_poll_interval` | duration | `10s` | `false` | Minimum interval between API polls. The receiver resets to this interval when data is received. Increase this to prevent hitting API rate limits. |
| `max_poll_interval` | duration | `5m` | `false` | Maximum interval between API polls. The receiver uses adaptive polling that starts at `min_poll_interval` and backs off when no data is returned, up to this maximum. |
| `backoff_multiplier` | float | `2.0` | `false` | Multiplier for increasing the poll interval when no data or a partial page is returned. Must be greater than 1.0. |
| `storage` | component | | `false` | The component ID of a storage extension for checkpointing |
| `timeout` | duration | `10s` | `false` | HTTP client timeout |

### Auth Mode Configuration

#### None (No Authentication)

Use `auth_mode: none` for public APIs that don't require authentication. No additional configuration is needed.

#### API Key

| Field | Type | Default | Required | Description |
|-------|------|---------|----------|-------------|
| `header_name` | string | | `true` | Header name for API key (required if `auth_mode` is `apikey`) |
| `value` | string | | `true` | API key value (required if `auth_mode` is `apikey`) |

#### Bearer Token

| Field | Type | Default | Required | Description |
|-------|------|---------|----------|-------------|
| `token` | string | | `true` | Bearer token value (required if `auth_mode` is `bearer`) |

#### Basic Auth

| Field | Type | Default | Required | Description |
|-------|------|---------|----------|-------------|
| `username` | string | | `true` | Username for basic auth (required if `auth_mode` is `basic`) |
| `password` | string | | `true` | Password for basic auth |

#### OAuth2 Client Credentials

| Field | Type | Default | Required | Description |
|-------|------|---------|----------|-------------|
| `client_id` | string | | `true` | OAuth2 client ID (required if `auth_mode` is `oauth2`) |
| `client_secret` | string | | `true` | OAuth2 client secret (required if `auth_mode` is `oauth2`) |
| `token_url` | string | | `true` | OAuth2 token endpoint URL (required if `auth_mode` is `oauth2`) |
| `scopes` | []string | | `false` | OAuth2 scopes to request |
| `endpoint_params` | map[string]string | | `false` | Additional parameters to send to the token endpoint |

#### Akamai EdgeGrid

**The Akamai API requires an enterprise license. This authentication method has not been tested against an Akamai API.**

| Field | Type | Default | Required | Description |
|-------|------|---------|----------|-------------|
| `akamai_access_token` | string || `true` | Akamai EdgeGrid access token |
| `akamai_client_token` | string || `true` | Akamai EdgeGrid client token |
| `akamai_client_secret` | string || `true` | Akamai EdgeGrid client secret |

### Pagination Configuration

| Field | Type | Default | Required | Description |
|-------|------|---------|----------|-------------|
| `pagination.mode` | string | `none` | `false` | Pagination mode: `none`, `offset_limit`, `page_size`, or `timestamp` |
| `pagination.total_record_count_field` | string | | `false` | Field name in response containing total record count |
| `pagination.page_limit` | int | `0` | `false` | Maximum number of pages to fetch (0 = no limit) |
| `pagination.zero_based_index` | bool | `false` | `false` | Indicates that the requested data starts at index 0 |

#### Offset/Limit Pagination

| Field | Type | Default | Required | Description |
|-------|------|---------|----------|-------------|
| `pagination.offset_limit.offset_field_name` | string | | `false` | Query parameter name for offset |
| `pagination.offset_limit.limit_field_name` | string | | `false` | Query parameter name for limit |
| `pagination.offset_limit.starting_offset` | int | `0` | `false` | Starting offset value |

#### Page/Size Pagination

| Field | Type | Default | Required | Description |
|-------|------|---------|----------|-------------|
| `pagination.page_size.page_num_field_name` | string | | `false` | Query parameter name for page number |
| `pagination.page_size.page_size_field_name` | string | | `false` | Query parameter name for page size |
| `pagination.page_size.starting_page` | int | `1` | `false` | Starting page number |
| `pagination.page_size.total_pages_field_name` | string | | `false` | Field name in response containing total page count |

#### Timestamp-Based Pagination

| Field | Type | Default | Required | Description |
|-------|------|---------|----------|-------------|
| `pagination.timestamp.param_name` | string | | `true` | Query parameter name for timestamp (e.g., "t0", "since", "after", "start_time") |
| `pagination.timestamp.timestamp_field_name` | string | | `true` | Field name in each response item containing the timestamp (e.g., "ts", "timestamp") |
| `pagination.timestamp.timestamp_format` | string | RFC3339 | `false` | Go time format string for the timestamp query parameter (e.g., "20060102150405" for YYYYMMDDHHMMSS) |
| `pagination.timestamp.page_size_field_name` | string | | `false` | Query parameter name for page size (e.g., "perPage", "limit") |
| `pagination.timestamp.page_size` | int | `100` | `false` | Page size to use |
| `pagination.timestamp.initial_timestamp` | string | | `false` | Initial timestamp to start from (RFC3339 format). If not set, starts from beginning |

Common timestamp formats:
- `2006-01-02T15:04:05Z07:00` - RFC3339 (default)
- `20060102150405` - YYYYMMDDHHMMSS
- `2006-01-02 15:04:05` - Date and time with space separator
- `2006-01-02` - Date only

### Metrics Configuration

The metrics configuration allows you to customize how metrics are extracted from API responses.

| Field | Type | Default | Required | Description |
|-------|------|---------|----------|-------------|
| `metrics.name_field` | string | | `true` | Field name in each response item containing the metric name. If not found, the metric will be dropped and a warning will be logged. |
| `metrics.description_field` | string | | `false` | Field name in each response item containing the metric description. If not specified or not found, defaults to `Metric from REST API` |
| `metrics.type_field` | string | | `false` | Field name in each response item containing the metric type (`gauge`, `sum`, `histogram`, `summary`). If not specified or not found, defaults to `gauge` |
| `metrics.unit_field` | string | | `false` | Field name in each response item containing the metric unit. If not specified or not found, no unit is set |
| `metrics.monotonic_field` | string | | `false` | Field name in each response item indicating if a sum metric is monotonic (boolean). Only applies to `sum` metrics. If not specified or not found, defaults to `false` for safety |
| `metrics.aggregation_temporality_field` | string | | `false` | Field name in each response item containing the aggregation temporality (`cumulative` or `delta`). If not specified or not found, defaults to `cumulative` |

**Note:** When field names are configured, those fields are automatically excluded from metric attributes to avoid duplication. If the `name_field` isn't found, the metric will be dropped.

## Example Configurations

### Basic Configuration (No Auth, No Pagination)

```yaml
receivers:
  restapi:
    url: "https://api.example.com/data"
    max_poll_interval: 5m
```

### API Key Authentication

```yaml
receivers:
  restapi:
    url: "https://api.example.com/events"
    max_poll_interval: 10m
    auth_mode: apikey
    apikey:
      header_name: "X-API-Key"
      value: "your-api-key-here"
```

### Bearer Token Authentication

```yaml
receivers:
  restapi:
    url: "https://api.example.com/metrics"
    max_poll_interval: 5m
    auth_mode: bearer
    bearer:
      token: "your-bearer-token-here"
```

### Basic Authentication with Pagination

```yaml
receivers:
  restapi:
    url: "https://api.example.com/logs"
    response_field: "data"
    max_poll_interval: 5m
    auth_mode: basic
    basic:
      username: "user"
      password: "pass"
    pagination:
      mode: offset_limit
      offset_limit:
        offset_field_name: "offset"
        limit_field_name: "limit"
        starting_offset: 0
      total_record_count_field: "total"
    storage: file_storage
```

### OAuth2 Client Credentials Authentication

```yaml
receivers:
  restapi:
    url: "https://api.example.com/data"
    max_poll_interval: 5m
    auth_mode: oauth2
    oauth2:
      client_id: "your-client-id"
      client_secret: "your-client-secret"
      token_url: "https://oauth.example.com/token"
```

### OAuth2 with Scopes and Custom Parameters

```yaml
receivers:
  restapi:
    url: "https://api.example.com/data"
    max_poll_interval: 5m
    auth_mode: oauth2
    oauth2:
      client_id: "your-client-id"
      client_secret: "your-client-secret"
      token_url: "https://oauth.example.com/token"
      scopes:
        - "read"
        - "write"
      endpoint_params:
        audience: "https://api.example.com"
        resource: "https://api.example.com"
```

### Akamai EdgeGrid Authentication

```yaml
receivers:
  restapi:
    url: "https://api.akamai.com/endpoint"
    max_poll_interval: 5m
    auth_mode: akamai_edgegrid
    akamai_edgegrid:
      access_token: "your-access-token"
      client_token: "your-client-token"
      client_secret: "your-client-secret"
```

### Timestamp Pagination

```yaml
receivers:
  restapi:
    url: "https://api.example.com/events"
    response_field: "items"
    max_poll_interval: 15m
    auth_mode: bearer
    bearer:
      token: "token"
    pagination:
      mode: timestamp
      timestamp:
        param_name: "t0"
        timestamp_field_name: "ts"
        page_size_field_name: "perPage"
        page_size: 200
        initial_timestamp: "2024-01-01T00:00:00Z"
    storage: file_storage

extensions:
  file_storage:
    directory: /var/lib/otelcol/storage
```

### Timestamp Pagination with Custom Format

Some APIs require specific timestamp formats. Use `timestamp_format` to specify the Go time format string:

```yaml
receivers:
  restapi:
    url: "https://api.example.com/events"
    response_field: "events"
    max_poll_interval: 10s
    auth_mode: none
    pagination:
      mode: timestamp
      timestamp:
        param_name: "min-date"
        timestamp_field_name: "timestamp"
        timestamp_format: "20060102150405"  # YYYYMMDDHHMMSS format
        initial_timestamp: "2025-01-01T00:00:00Z"
```

### Metrics with Custom Field Mappings

```yaml
receivers:
  restapi/metrics:
    url: "https://api.example.com/metrics"
    response_field: "metrics"
    max_poll_interval: 1m
    auth_mode: apikey
    apikey:
      header_name: "X-API-Key"
      value: "your-api-key-here"
    metrics:
      name_field: "metric_name"
      description_field: "metric_description"
      type_field: "metric_type"
      unit_field: "unit"
      monotonic_field: "is_monotonic"
      aggregation_temporality_field: "aggregation"

service:
  pipelines:
    metrics:
      receivers: [restapi/metrics]
      exporters: [otlp]
```

This configuration would work with an API response like:
```json
{
  "metrics": [
    {
      "metric_name": "cpu.usage",
      "metric_description": "CPU usage percentage",
      "metric_type": "gauge",
      "unit": "%",
      "value": 45.5,
      "host": "server1",
      "environment": "production"
    },
    {
      "metric_name": "http.requests.total",
      "metric_description": "Total HTTP requests",
      "metric_type": "sum",
      "unit": "requests",
      "is_monotonic": true,
      "aggregation": "cumulative",
      "value": 8589934592,
      "host": "server1",
      "environment": "production"
    }
  ]
}
```

## Response Format

The receiver expects JSON responses in one of two formats:

1. **Top-level array:**
```json
[
  {"id": "1", "message": "log entry 1"},
  {"id": "2", "message": "log entry 2"}
]
```

2. **Object with data field:**
```json
{
  "data": [
    {"id": "1", "message": "log entry 1"},
    {"id": "2", "message": "log entry 2"}
  ],
  "total": 2
}
```

When using the second format, specify the field name in `response_field` (e.g., `"data"`).

## Adaptive Polling

The receiver uses adaptive polling to balance responsiveness with API rate limits. Instead of polling at a fixed interval, it adjusts the poll interval based on whether data is being returned.

- **On startup**, the receiver polls at `min_poll_interval`.
- **When a full page is returned** (indicating more data may be waiting), the interval resets to `min_poll_interval` to fetch remaining data quickly.
- **When no data or a partial page is returned** (indicating the receiver is caught up), the interval is multiplied by `backoff_multiplier` each cycle, up to `max_poll_interval`.

For example, with the defaults (`min_poll_interval: 10s`, `max_poll_interval: 5m`, `backoff_multiplier: 2.0`), the polling intervals when no new data arrives would be: 10s, 20s, 40s, 80s, 160s, 300s (capped at 5m). As soon as a full page of data is returned, the interval resets back to 10s.

To poll at a fixed interval, set `min_poll_interval` and `max_poll_interval` to the same value.

## Checkpointing

When a storage extension is configured, the receiver saves its pagination state to storage. This allows the receiver to resume from where it left off after a restart, preventing duplicate data collection.

The checkpoint includes:
- Current pagination state (offset/page number/timestamp)
- Number of pages fetched

For timestamp-based pagination, the timestamp is reset after each poll cycle to the initial timestamp, ensuring each poll starts fresh and only collects new data based on the time filter.
