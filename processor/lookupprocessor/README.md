# Lookup Processor
This processor is used to lookup values from multiple data sources including CSV files, Redis databases, and REST APIs. Lookups can be cached locally to improve performance and minimize calls to external sources.

## Supported pipelines
- Logs
- Metrics
- Traces

## How It Works
1. The processor connects to a configured data source (CSV file, Redis, or REST API).
2. When telemetry is received, the processor checks if the configured `field` exists in the configured `context`.
3. If the field exists, the processor looks up the value in the data source.
4. If caching is enabled (default), the processor first checks the local cache before making external calls.
5. All matching fields from the lookup are added to the `context` of the telemetry.

## Configuration

### Common Fields
| Field          | Type     | Default | Required | Description |
| ---            | ---      | ---     | ---      | ---         |
| context        | string   | ` `     | Yes      | The context of the telemetry to check and use when performing lookups. Supported values are `attributes`, `body`, `resource.attributes`. |
| field          | string   | ` `     | Yes      | The field to match when performing a lookup. For a lookup to succeed, the field name must exist in the telemetry. |
| source_type    | string   | `csv`   | No       | The type of lookup source. Supported values are `csv`, `redis`, `api`. If not specified and `csv` is provided, defaults to `csv`. |

### Cache Configuration
| Field            | Type          | Default | Description |
| ---              | ---           | ---     | ---         |
| cache_enabled    | boolean       | `true`  | Whether to enable local caching of lookup results. |
| cache_ttl        | duration      | `5m`    | How long cached entries remain valid before requiring a refresh. |
| cache_storage_id | component.ID  | ` `     | Optional reference to a storage extension for shared caching across processor instances. If not specified, uses processor-local storage. |

### CSV Source Configuration
| Field | Type   | Default | Description |
| ---   | ---    | ---     | ---         |
| csv   | string | ` `     | The location of the CSV file used for lookups. The processor will periodically reload this in memory every minute. |

### Redis Source Configuration
| Field            | Type    | Default | Description |
| ---              | ---     | ---     | ---         |
| redis.address    | string  | ` `     | Redis server address (e.g., `localhost:6379`). |
| redis.username   | string  | ` `     | Redis username for authentication. |
| redis.password   | string  | ` `     | Redis password for authentication. |
| redis.db         | int     | `0`     | Redis database number to use. |
| redis.tls        | boolean | `false` | Whether to use TLS for the Redis connection. |
| redis.key_prefix | string  | ` `     | Optional prefix to add to lookup keys in Redis. |

**Redis Data Format:**
The Redis source supports two data formats:
1. **Redis Hash (recommended)**: Store data as a Redis hash where the hash fields are the attribute names
   ```
   HSET myhost hostname "host-1" region "us-west" env "prod"
   ```
2. **JSON String**: Store data as a JSON object
   ```
   SET myhost '{"hostname":"host-1","region":"us-west","env":"prod"}'
   ```

### API Source Configuration
| Field                     | Type              | Default | Description |
| ---                       | ---               | ---     | ---         |
| api.url                   | string            | ` `     | The API endpoint URL. Supports template variables `$fieldValue`, `${fieldValue}`, `$key`, `${key}` which are replaced with the lookup key value. |
| api.method                | string            | `GET`   | HTTP method to use for API requests. |
| api.headers               | map[string]string | ` `     | HTTP headers to include in API requests. Use for authentication (e.g., `Authorization: Bearer token`, `X-API-Key: key`). |
| api.timeout               | duration          | `30s`   | Request timeout for API calls. |
| api.response_mapping      | map[string]string | ` `     | Optional mapping to extract specific fields from JSON response. If not specified, all top-level fields from the JSON response are used. Format: `new_field_name: json.path.to.field`. |

## Example Configurations

### Example 1: CSV Source (Backward Compatible)
The following is an example configuration using a CSV file. This processor will check if incoming logs contain an `ip` field on their body. If they do, the processor will use the value of `ip` to lookup additional fields in the `example.csv` file. If a match is found, all other defined values in the csv will be added to the body of the log.

```yaml
receivers:
    otlp:
        protocols:
            grpc:
processors:
    lookup:
        csv: ./example.csv
        context: body
        field: ip
        # Optional: Configure caching
        cache_enabled: true
        cache_ttl: 10m
exporters:
    logging:
service:
    pipelines:
        logs:
            receivers: [otlp]
            processors: [lookup]
            exporters: [logging]
```

**Example CSV file (example.csv):**
```csv
ip,host,region,env
0.0.0.0,host-1,us-west,prod
1.1.1.1,host-2,us-east,dev
```

### Example 2: Redis Source
Lookup host information from Redis:

```yaml
processors:
    lookup:
        source_type: redis
        context: attributes
        field: hostname
        redis:
            address: localhost:6379
            password: ${env:REDIS_PASSWORD}
            db: 0
            key_prefix: "hosts"
            tls: false
        cache_enabled: true
        cache_ttl: 5m
```

**Example Redis data:**
```bash
# Using Redis Hash (recommended)
HSET hosts:server1 region "us-west" datacenter "dc1" owner "team-platform"
HSET hosts:server2 region "us-east" datacenter "dc2" owner "team-infra"

# Or using JSON strings
SET hosts:server1 '{"region":"us-west","datacenter":"dc1","owner":"team-platform"}'
```

### Example 3: REST API Source
Lookup enrichment data from an API endpoint:

```yaml
processors:
    lookup:
        source_type: api
        context: resource.attributes
        field: service_name
        api:
            url: https://metadata-service.example.com/api/v1/services/$fieldValue
            method: GET
            headers:
                Authorization: Bearer ${env:API_TOKEN}
                X-API-Key: ${env:API_KEY}
            timeout: 15s
        cache_enabled: true
        cache_ttl: 15m
```

### Example 4: API Source with Response Mapping
Extract specific fields from a nested JSON API response:

```yaml
processors:
    lookup:
        source_type: api
        context: attributes
        field: user_id
        api:
            url: https://api.example.com/users/$fieldValue
            method: GET
            headers:
                X-API-Key: ${env:API_KEY}
            response_mapping:
                # Map API response fields to attribute names
                user_email: data.email
                user_department: metadata.department
                user_location: metadata.office.location
        cache_enabled: true
        cache_ttl: 30m
```

**Example API Response:**
```json
{
    "data": {
        "email": "user@example.com",
        "name": "John Doe"
    },
    "metadata": {
        "department": "Engineering",
        "office": {
            "location": "San Francisco"
        }
    }
}
```

The processor will extract and add: `user_email="user@example.com"`, `user_department="Engineering"`, `user_location="San Francisco"`.

### Example 5: Shared Cache with File Storage Extension
Use a shared cache across multiple lookup processor instances:

```yaml
extensions:
    file_storage:
        directory: /var/lib/otelcol/storage

processors:
    lookup/service:
        source_type: api
        context: attributes
        field: service_id
        api:
            url: https://api.example.com/services/$fieldValue
            headers:
                Authorization: Bearer ${env:API_TOKEN}
        cache_enabled: true
        cache_ttl: 1h
        cache_storage_id: file_storage

    lookup/user:
        source_type: redis
        context: attributes
        field: user_id
        redis:
            address: redis:6379
        cache_enabled: true
        cache_ttl: 30m
        cache_storage_id: file_storage

service:
    extensions: [file_storage]
    pipelines:
        logs:
            receivers: [otlp]
            processors: [lookup/service, lookup/user]
            exporters: [logging]
```
