# REST API Receiver

The REST API receiver is a generic receiver that can pull data from any REST API endpoint. It supports both logs and metrics collection, with configurable authentication, pagination, and time-based offset tracking.

## Supported Pipelines

- Logs
- Metrics

## How It Works

1. The receiver polls a configured REST API endpoint at a specified interval.
2. It handles authentication (API Key, Bearer Token, Basic Auth, or Akamai EdgeGrid).
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
| `auth_mode` | string | `apikey` | `false` | Authentication mode: `apikey`, `bearer`, `basic`, `oauth2`, or `akamai_edgegrid` |
| `apikey` | object | | `false` | API Key configuration (see below) |
| `bearer` | object | | `false` | Bearer Token configuration (see below) |
| `basic` | object | | `false` | Basic Auth configuration (see below) |
| `oauth2` | object | | `false` | OAuth2 Client Credentials configuration (see below) |
| `akamai_edgegrid` | object | | `false` | Akamai EdgeGrid configuration (see below) |
| `pagination` | object | | `false` | Pagination configuration (see below) |
| `poll_interval` | duration | `5m` | `false` | The interval between API polls |
| `storage` | component | | `false` | The component ID of a storage extension for checkpointing |
| `timeout` | duration | `10s` | `false` | HTTP client timeout |

### Auth Mode Configuration

#### API Key

| Field | Type | Default | Required | Description |
|-------|------|---------|----------|-------------|
| `apikey_header_name` | string | | `true` | Header name for API key (required if `auth_mode` is `apikey`) |
| `apikey_value` | string | | `true` | API key value (required if `auth_mode` is `apikey`) |

#### Bearer Token

| Field | Type | Default | Required | Description |
|-------|------|---------|----------|-------------|
| `bearer_token` | string | | `true` | Bearer token value (required if `auth_mode` is `bearer`) |

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
| `pagination.timestamp.page_size_field_name` | string | | `true` | Query parameter name for page size (e.g., "perPage", "limit") |
| `pagination.timestamp.page_size` | int | `100` | `false` | Page size to use |
| `pagination.timestamp.initial_timestamp` | string | | `false` | Initial timestamp to start from (RFC3339 format). If not set, starts from beginning |

### Metrics Configuration

The metrics configuration allows you to customize how metrics are extracted from API responses.

| Field | Type | Default | Required | Description |
|-------|------|---------|----------|-------------|
| `metrics.name_field` | string | | `false` | Field name in each response item containing the metric name. If not specified or not found, defaults to `restapi.metric` |
| `metrics.description_field` | string | | `false` | Field name in each response item containing the metric description. If not specified or not found, defaults to `Metric from REST API` |
| `metrics.type_field` | string | | `false` | Field name in each response item containing the metric type (`gauge`, `sum`, `histogram`, `summary`). If not specified or not found, defaults to `gauge` |
| `metrics.unit_field` | string | | `false` | Field name in each response item containing the metric unit. If not specified or not found, no unit is set |
| `metrics.monotonic_field` | string | | `false` | Field name in each response item indicating if a sum metric is monotonic (boolean). Only applies to `sum` metrics. If not specified or not found, defaults to `false` for safety |
| `metrics.aggregation_temporality_field` | string | | `false` | Field name in each response item containing the aggregation temporality (`cumulative` or `delta`). If not specified or not found, defaults to `cumulative` |

**Note:** When field names are configured, those fields are automatically excluded from metric attributes to avoid duplication.

## Example Configurations

### Basic Configuration (No Auth, No Pagination)

```yaml
receivers:
  restapi:
    url: "https://api.example.com/data"
    poll_interval: 5m
```

### API Key Authentication

```yaml
receivers:
  restapi:
    url: "https://api.example.com/events"
    poll_interval: 10m
    auth_mode: apikey
    apikey_header_name: "X-API-Key"
    apikey_value: "your-api-key-here"
```

### Bearer Token Authentication

```yaml
receivers:
  restapi:
    url: "https://api.example.com/metrics"
    poll_interval: 5m
    auth_mode: bearer
    bearer_token: "your-bearer-token-here"
```

### Basic Authentication with Pagination

```yaml
receivers:
  restapi:
    url: "https://api.example.com/logs"
    response_field: "data"
    poll_interval: 5m
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
    poll_interval: 5m
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
    poll_interval: 5m
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
    poll_interval: 5m
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
    poll_interval: 15m
    auth_mode: bearer
    bearer_token: "token"
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

### Metrics with Custom Field Mappings

```yaml
receivers:
  restapi/metrics:
    url: "https://api.example.com/metrics"
    response_field: "metrics"
    poll_interval: 1m
    auth_mode: apikey
    apikey_header_name: "X-API-Key"
    apikey_value: "your-api-key-here"
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

## Checkpointing

When a storage extension is configured, the receiver saves its pagination state to storage. This allows the receiver to resume from where it left off after a restart, preventing duplicate data collection.

The checkpoint includes:
- Current pagination state (offset/page number/timestamp)
- Number of pages fetched

For timestamp-based pagination, the timestamp is reset after each poll cycle to the initial timestamp, ensuring each poll starts fresh and only collects new data based on the time filter.
