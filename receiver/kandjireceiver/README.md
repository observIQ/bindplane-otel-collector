# Kandji Receiver

Receives logs from [Kandji](https://www.kandji.com/) via the [Kandji API](https://docs.kandji.com/api-docs).

## Overview

The Kandji receiver collects audit event logs from Kandji's MDM (Mobile Device Management) platform. It uses a flexible registry-based pattern to handle Kandji's API endpoints with automatic pagination, cursor-based checkpointing, and comprehensive parameter validation.

## Supported Pipelines

- Logs

## Prerequisites

- A Kandji account with API access
- A Kandji API key/token
- Your Kandji subdomain (e.g., "acme" in `https://acme.kandji.io`)

## Configuration

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `subdomain` | string | `(no default)` | **Required.** Your Kandji subdomain (e.g., "acme" for `https://acme.kandji.io`) |
| `api_key` | string | `(no default)` | **Required.** Your Kandji API token/key |
| `region` | string | `"US"` | Kandji region: `"US"` or `"EU"` |
| `base_host` | string | `"api.kandji.io"` | Base hostname for the Kandji API (usually not needed) |
| `logs` | object | `(n/a)` | Configuration object for logs collection |
| `logs.poll_interval` | duration | `5m` | Interval at which to poll for new logs (e.g., `5m`, `10m`, `1h`) |
| `storage_id` | component | `(no default)` | **Recommended.** Component ID of a storage extension to persist cursors across restarts. Prevents duplicate log collection after agent restarts. |

### HTTP Client Configuration

The receiver uses the standard OpenTelemetry `confighttp.ClientConfig` for HTTP client settings:

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `timeout` | duration | `15s` | HTTP request timeout |

## Example Configurations

### Basic Configuration (Logs Only)

```yaml
receivers:
  kandji:
    subdomain: "acme"
    api_key: "${KANDJI_API_KEY}"
    logs:
      poll_interval: 5m
    storage_id: file_storage

extensions:
  file_storage:
    directory: /var/lib/otelcol/storage

exporters:
  file/no_rotation:
    path: /tmp/kandji-logs.json

service:
  extensions: [file_storage]
  pipelines:
    logs:
      receivers: [kandji]
      exporters: [file/no_rotation]
```

### EU Region Configuration

```yaml
receivers:
  kandji:
    subdomain: "acme"
    region: "EU"
    api_key: "${KANDJI_API_KEY}"
    logs:
      poll_interval: 10m
    storage_id: file_storage
```

### Custom Poll Interval

```yaml
receivers:
  kandji:
    subdomain: "acme"
    api_key: "${KANDJI_API_KEY}"
    logs:
      poll_interval: 15m
    storage_id: file_storage
```

## How It Works

1. **Configuration**: The receiver is configured with your Kandji subdomain and API key
2. **Polling**: At the configured interval, the receiver polls the Kandji API for new audit events
3. **Pagination**: The receiver automatically handles cursor-based pagination to collect all available events
4. **Checkpointing**: Cursors are saved to the storage extension to resume from the last collected position after restarts
5. **Emission**: Audit events are transformed into OpenTelemetry log records and sent to downstream consumers

## Data Collected

### Audit Events

The receiver collects audit event logs from Kandji, which include:
- **Security events**: Login attempts, authentication changes
- **Administrative actions**: Device enrollment, policy changes, user management
- **Device events**: Device status changes, compliance updates

Each log record contains:
- Full event JSON in the log body
- Event metadata as log attributes:
  - `id`: Event ID
  - `action`: Action type
  - `actor.id`: Actor ID
  - `actor.type`: Actor type
  - `target.id`: Target ID
  - `target.type`: Target type
  - `target.component`: Target component
- Resource attributes:
  - `kandji.region`: Kandji region (US/EU)
  - `kandji.subdomain`: Kandji subdomain
  - `kandji.endpoint`: API endpoint used
  - `kandji.log_type`: Log type (currently "audit_events")

## Important Notes

- **Storage Extension**: It is highly recommended to configure a `storage_id` to persist cursors across collector restarts. Without storage, the receiver will start from the beginning on each restart, potentially collecting duplicate logs.

- **Poll Interval**: The default poll interval is 5 minutes. Adjust based on your needs:
  - More frequent polling (e.g., `1m`) provides near-real-time logs but increases API usage
  - Less frequent polling (e.g., `30m`) reduces API usage but increases latency

- **API Rate Limits**: Be aware of Kandji's API rate limits. The receiver includes automatic pagination and respects API responses, but very frequent polling may hit rate limits.

- **Initial Collection**: On first run, the receiver will collect all available audit events from the API. This may take some time depending on your account's event history.

## Getting a Kandji API Key

1. Log in to your Kandji instance at `https://{subdomain}.kandji.io`
2. Navigate to **Settings** → **API**
3. Generate a new API token
4. Copy the token value (it will only be shown once)
5. Use this token as the `api_key` in your receiver configuration

For more information, see the [Kandji API documentation](https://docs.kandji.com/api-docs).

## Troubleshooting

### No Logs Appearing

- Verify your `subdomain` and `api_key` are correct
- Check that the collector has network access to `https://{subdomain}.api.kandji.io`
- Review collector logs for API errors
- Ensure the storage extension is working if using `storage_id`

### Duplicate Logs

- Ensure `storage_id` is configured and the storage extension is working
- Check that cursors are being persisted (look for checkpoint save/load logs)

### API Errors

- Verify your API key is valid and has not expired
- Check that your account has API access enabled
- Review the error messages in collector logs for specific API error details

---

# Architecture

## Overview

The Kandji receiver is an OpenTelemetry Collector receiver that collects **logs** from the Kandji API. It implements a flexible registry-based pattern to handle Kandji's non-standardized API endpoints, each with unique parameter requirements, validation rules, and pagination strategies.

## Registry Pattern

### Problem Statement

Kandji's API endpoints are not standardized. Different endpoints have:
- **Different parameter types** (strings, ints, enums, timestamps, UUIDs)
- **Different validation rules** (min/max values, allowed enums, time constraints)
- **Different pagination strategies** (cursor-based, offset-based, or none)
- **Different response structures** (various JSON shapes)

A traditional approach of hardcoding each endpoint would lead to:
- Code duplication
- Difficult maintenance
- Error-prone parameter handling
- Inconsistent validation

### Solution: Declarative Endpoint Registry

The receiver uses a **declarative registry pattern** where each API endpoint is registered with its complete specification:

```go
EndpointRegistry[EPAuditEventsList] = EndpointSpec{
    Method:             "GET",
    Path:               "/audit/events",
    Description:        "List audit log events.",
    SupportsPagination: true,
    Params: []ParamSpec{
        {
            Name: "limit",
            Type: ParamInt,
            Constraints: &ParamConstraints{
                MaxInt: &maxLimit,  // limit <= 500
            },
        },
        {
            Name:        "sort_by",
            Type:        ParamString,
            AllowedVals: []string{"occurred_at", "-occurred_at"},  // enum validation
        },
        CursorParam,  // reusable cursor parameter
    },
    ResponseType: AuditEventsResponse{},
}
```

### Key Components

#### 1. **EndpointSpec** - Complete endpoint definition
- `Method`: HTTP method (GET, POST, etc.)
- `Path`: API path
- `Params`: List of parameter specifications
- `ResponseType`: Expected response structure
- `SupportsPagination`: Whether endpoint supports cursor-based pagination

#### 2. **ParamSpec** - Parameter validation rules
- `Name`: Parameter name
- `Type`: Data type (String, Int, Bool, Time, Enum, UUID, Float)
- `Required`: Whether parameter is mandatory
- `AllowedVals`: For enum types, list of valid values
- `Constraints`: Type-specific validation rules

#### 3. **ParamConstraints** - Type-specific validation
- **Numbers**: `MinInt`, `MaxInt`
- **Strings**: `MinLen`, `MaxLen`
- **Times**: `NotBefore`, `NotAfter`, `MaxAge`, `NotNewerThanNow`

#### 4. **ValidateParams()** - Runtime validation
Validates user-provided parameters against the endpoint's spec:
- Type checking
- Constraint validation
- Required parameter checking
- Enum value validation

### Benefits

1. **Type Safety**: Compile-time checking of endpoint definitions
2. **Centralized Validation**: Single validation function handles all endpoints
3. **Easy Extension**: Adding new endpoints requires only registry entry
4. **Self-Documenting**: Registry serves as API documentation
5. **Consistent Error Messages**: Standardized validation errors
6. **Reusable Components**: Common params (like `CursorParam`) can be shared

### Example: Adding a New Endpoint

```go
// 1. Define endpoint constant
const EPDevicesList KandjiEndpoint = "GET /devices"

// 2. Register in init()
EndpointRegistry[EPDevicesList] = EndpointSpec{
    Method:             "GET",
    Path:               "/devices",
    SupportsPagination: false,
    Params: []ParamSpec{
        {Name: "status", Type: ParamEnum, AllowedVals: []string{"active", "inactive"}},
    },
    ResponseType: DevicesResponse{},
}

// 3. That's it! The logs receiver automatically handles it
```

## Receiver Description

### Purpose

The Kandji receiver enables collection of telemetry data from Kandji's MDM (Mobile Device Management) platform, specifically:
- **Audit Events**: Security and administrative activity logs

### Features

1. **Logs Collection**
   - Collects audit event logs with cursor-based checkpointing

2. **Flexible Configuration**
   - Configurable polling intervals
   - Region support (US/EU)
   - Custom base host support

3. **Robust Pagination**
   - Automatic cursor-based pagination
   - Cursor normalization and sanitization
   - Checkpoint persistence for logs receiver

4. **Comprehensive Validation**
   - Parameter type checking
   - Constraint validation
   - Enum value validation
   - Time range validation

5. **Observability**
   - Detailed debug logging
   - Request/response logging
   - Error tracking

## Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────┐
│                    OpenTelemetry Collector                      │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                    Kandji Receiver Factory                       │
│  ┌──────────────────────┐                                        │
│  │ Logs Receiver         │                                        │
│  │ (Polling-based)       │                                        │
│  └──────────────────────┘                                        │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                    Endpoint Registry                             │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │  EndpointRegistry[EPAuditEventsList] = EndpointSpec {     │  │
│  │    Method: "GET"                                          │  │
│  │    Path: "/audit/events"                                 │  │
│  │    Params: [limit, sort_by, cursor]                      │  │
│  │    SupportsPagination: true                             │  │
│  │    ResponseType: AuditEventsResponse                     │  │
│  │  }                                                       │  │
│  └──────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                    Parameter Validation                          │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │  ValidateParams(endpoint, params)                       │  │
│  │    ├─ Type checking (int, string, enum, time, etc.)    │  │
│  │    ├─ Constraint validation (min/max, ranges)          │  │
│  │    └─ Required parameter checking                      │  │
│  └──────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                    Kandji HTTP Client                            │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │  CallAPI(endpoint, params, response)                     │  │
│  │    ├─ BuildURL() - Constructs URL with query params    │  │
│  │    ├─ HTTP Request with Bearer auth                     │  │
│  │    ├─ Response parsing                                  │  │
│  │    └─ Error handling                                    │  │
│  └──────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                    Kandji API                                    │
│              https://{subdomain}.api.kandji.io/api/v1          │
└─────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────┐
│                    Logs Flow                                     │
│  ┌──────────────┐    ┌──────────────┐    ┌──────────────┐      │
│  │  Poll Loop   │───▶│ pollEndpoint │───▶│  emitLogs()  │      │
│  │  (ticker)    │    │ (paginated)  │    │ (to logs)    │      │
│  └──────────────┘    └──────────────┘    └──────────────┘      │
│         │                    │                    │              │
│         │                    ▼                    │              │
│         │         ┌──────────────────┐           │              │
│         └────────▶│  Checkpoint     │◀──────────┘              │
│                   │  (cursor save)   │                          │
│                   └──────────────────┘                          │
└─────────────────────────────────────────────────────────────────┘
```

### Component Details

#### Logs Receiver (Polling)
- **Trigger**: Polling interval (default: 5 minutes)
- **Purpose**: Collect audit event logs
- **Flow**: `pollAll()` → `pollEndpoint()` → `fetchPage()` → `emitLogs()`
- **Checkpointing**: Cursors saved to storage extension for resumability
- **Output**: OpenTelemetry logs (audit events as JSON)

#### Shared Components
- **Endpoint Registry**: Declarative endpoint definitions
- **Parameter Validator**: Runtime validation against specs
- **HTTP Client**: Unified API client with auth and error handling
- **Cursor Normalization**: Handles various cursor formats (URLs, query strings, plain strings)

### Data Flow

1. **Configuration** → User configures polling interval and storage
2. **Validation** → Parameters validated against endpoint specs
3. **Request** → HTTP client builds URL and makes authenticated request
4. **Response** → Response parsed into typed structures
5. **Pagination** → If paginated, cursor extracted and next page requested
6. **Transformation** → Data transformed to OTel log format
7. **Emission** → Logs sent to downstream consumers
8. **Checkpointing** → Cursor saved for next poll

### Error Handling

- **Validation Errors**: Caught before API calls
- **HTTP Errors**: Logged with status code and response body
- **Pagination Errors**: Stop pagination, return partial results
- **Transformation Errors**: Logged, individual records skipped

### Extensibility

Adding support for new Kandji endpoints requires:
1. Define response type (if new)
2. Add endpoint constant
3. Register in `EndpointRegistry` with full spec
4. No changes to logs receiver logic needed

This makes the receiver highly maintainable and easy to extend as Kandji adds new API endpoints.
