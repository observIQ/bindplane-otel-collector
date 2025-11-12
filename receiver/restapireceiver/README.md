# REST API Receiver

The REST API receiver is a generic receiver that can pull data from any REST API endpoint. It supports both logs and metrics collection, with configurable authentication, pagination, and time-based offset tracking.

## Supported Pipelines

- Logs
- Metrics

## How It Works

1. The receiver polls a configured REST API endpoint at a specified interval.
2. It handles authentication (API Key, Bearer Token, or Basic Auth).
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
| `response_field` | string | | `false` | The name of the field in the response that contains the array of items. If empty, the response is assumed to be a top-level array |
| `auth_mode` | string | `apikey` | `false` | Authentication mode: `apikey`, `bearer`, or `basic` |
| `apikey_header_name` | string | | `false` | Header name for API key (required if `auth_mode` is `apikey`) |
| `apikey_value` | string | | `false` | API key value (required if `auth_mode` is `apikey`) |
| `bearer_token` | string | | `false` | Bearer token value (required if `auth_mode` is `bearer`) |
| `username` | string | | `false` | Username for basic auth (required if `auth_mode` is `basic`) |
| `password` | string | | `false` | Password for basic auth (required if `auth_mode` is `basic`) |
| `pagination` | object | | `false` | Pagination configuration (see below) |
| `time_based_offset` | object | | `false` | Time-based offset configuration (see below) |
| `poll_interval` | duration | `5m` | `false` | The interval between API polls |
| `storage` | component | | `false` | The component ID of a storage extension for checkpointing |
| `timeout` | duration | `10s` | `false` | HTTP client timeout |

### Pagination Configuration

| Field | Type | Default | Required | Description |
|-------|------|---------|----------|-------------|
| `pagination.mode` | string | `none` | `false` | Pagination mode: `none`, `offset_limit`, or `page_size` |
| `pagination.total_record_count_field` | string | | `false` | Field name in response containing total record count |
| `pagination.page_limit` | int | `0` | `false` | Maximum number of pages to fetch (0 = no limit) |
| `pagination.zero_based_index` | bool | `false` | `false` | Whether pagination starts at 0 (for page/size mode) |

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
| `pagination.page_size.total_pages_field_name` | string | | `false` | Field name in response containing total pages |

### Time-Based Offset Configuration

| Field | Type | Default | Required | Description |
|-------|------|---------|----------|-------------|
| `time_based_offset.enabled` | bool | `false` | `false` | Enable time-based offset tracking |
| `time_based_offset.param_name` | string | | `false` | Query parameter name for timestamp (required if enabled) |
| `time_based_offset.offset_timestamp` | string | | `false` | Initial timestamp offset (RFC3339 format). If not set, defaults to poll_interval ago |

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

### Time-Based Offset with Page/Size Pagination

```yaml
receivers:
  restapi:
    url: "https://api.example.com/events"
    response_field: "items"
    poll_interval: 15m
    auth_mode: bearer
    bearer_token: "token"
    pagination:
      mode: page_size
      page_size:
        page_num_field_name: "page"
        page_size_field_name: "size"
        starting_page: 1
        total_pages_field_name: "total_pages"
      page_limit: 100
    time_based_offset:
      enabled: true
      param_name: "since"
      offset_timestamp: "2024-01-01T00:00:00Z"
    storage: file_storage

extensions:
  file_storage:
    directory: /var/lib/otelcol/storage
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

When a storage extension is configured, the receiver saves its pagination state and time offset to storage. This allows the receiver to resume from where it left off after a restart, preventing duplicate data collection.

The checkpoint includes:
- Current pagination state (offset/page number)
- Time-based offset timestamp
- Number of pages fetched
