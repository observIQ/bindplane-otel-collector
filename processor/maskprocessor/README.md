# Mask Processor
This processor is used to detect and mask sensitive data.

## Supported pipelines
- Logs
- Metrics
- Traces

## How It Works
1. This processor traverses the attribute and resource fields of incoming telemetry. For log-based telemetry, it will also traverse the body.
2. If a field matches a defined regex rule, the matching value is replaced with `[masked_value]`, where `value` is the name of the  rule. For instance, a rule that masks email addresses would result in `[masked_email]`.
3. If a field is to be excluded from masking, it can be specified in the processor's `exclude` field. By default, the processor will mask all fields.

**Note**: Only attributes that are strings will be considered for masking. For example the phone number as a string `"8881234"` can be masked but as an integer `8881234` will not be.

## Configuration
| Field        | Type     | Default | Description |
| ---          | ---      | ---     | ---         |
| rules        | map      | `email`: `\b[a-z0-9._%\+\-—\|]+@[a-z0-9.\-—\|]+\.[a-z\|]{2,6}\b`<br /><br />`ssn`: `\b\d{3}[- ]\d{2}[- ]\d{4}\b`<br /><br />`credit_card`: `\b(?:(?:(?:\d{4}[- ]?){3}\d{4}\|\d{15,16}))\b`<br /><br />`phone`: `\b((\+\|\b)[1l][\-\. ])?\(?\b[\dOlZSB]{3,5}([\-\. ]\|\) ?)[\dOlZSB]{3}[\-\. ][\dOlZSB]{4}\b`|     | A series of key value pairs that define the masking rules of the processor. The key is the name of the rule. The value is the regex to mask. The regex engine used is [standard golang](https://pkg.go.dev/regexp/syntax). |
| exclude      | []string | `[]`    | A list of json dot notation fields that will be excluded from masking. The prefixes `resource`, `attributes`, and `body` can be used to indicate the root of the field. |

### Example Config
The following config is an example configuration of the mask processor using default values. This configuration will receive logs through an otlp receiver. The mask processor will then search and mask any logs that match the predefined email, ssn, credit_card, or phone rules. The logs will then be sent to the logging exporter.
```yaml
receivers:
    otlp:
        protocols:
            grpc:
processors:
    mask:
exporters:
    logging:
service:
    pipelines:
        logs:
            receivers: [otlp]
            processors: [mask]
            exporters: [logging]
```

## How To
### Add custom rules
The following configuration adds a custom rule to mask any word greater than 10 characters. In this example, matching values will be replaced with `masked_long_word`.
```yaml
processors:
    mask:
        rules:
            long_word: '\w{10,}'
```
### Exclude specific fields
The following configuration excludes the resource attribute `ip` from masking. In this scenario, the user wants to avoid masking this value, because it's only related to infrastructure, rather than pii.
```yaml
processors:
    mask:
        exclude: [resource.ip]
        rules:
            ip: '(?:[0-9]{1,3}\.){3}[0-9]{1,3}'
```

### Exclude all values
The following configuration excludes all attributes and resources from masking. In this scenario, the user wants to only mask data in the body of their log.
```yaml
processors:
    mask:
        exclude: [resource, attributes]
        rules:
            ip: '(?:[0-9]{1,3}\.){3}[0-9]{1,3}'
```

## Common Rules
The following is a list of example regex patterns that are often used to detect sensitive information.

| Value         | Regex |
| ---           | ---   |
| `email`       | `\b[a-z0-9._%\+\-—\|]+@[a-z0-9.\-—\|]+\.[a-z\|]{2,6}\b` |
| `ssn`         | `\b\d{3}[- ]\d{2}[- ]\d{4}\b` |
| `credit_card` | `\b(?:(?:(?:\d{4}[- ]?){3}\d{4}\|\d{15,16}))\b` |
| `phone`       | `\b((\+\|\b)[1l][\-\. ])?\(?\b[\dOlZSB]{3,5}([\-\. ]\|\) ?)[\dOlZSB]{3}[\-\. ][\dOlZSB]{4}\b` |
| `ipv4`        | `(?:[0-9]{1,3}\.){3}[0-9]{1,3}` |
| `us_street`   | `\b\d{1,8}\b[\s\S]{10,100}?\b([A-Z]{2})\b\s\d{5}\b` |
| `date`        | `(\d{4}\|\d{1,2})[\/\-]\d{1,2}[\/\-](\d{4}\|\d{1,2})` |
