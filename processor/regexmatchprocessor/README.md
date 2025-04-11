# Regex Match Processor

This processor will evaluate each log record's body against a set of named regexes. If a match is found, the log record will be enriched with the corresponding regex name as an attribute.

## Supported pipelines

- Logs

## Configuration
| Field     | Type   | Default | Description                                                                                                |
| --------- | ------ | ------- | ---------------------------------------------------------------------------------------------------------- |
| attribute_name | string | log.type | The name of the log record attribute in which the matching regex name will be stored |
| regexes | []regexmatcher.NamedRegex | [] | An ordered list of regexes to evaluate |
| default_name | string | "" | (optional) If specified, this value will be applied when no regex matches the log |

### Note

The regexes are evaluated in the order they are defined. The first regex that matches will be used and no further regexes will be evaluated.


### Example configuration

```yaml
regexmatch:
    attribute_name: log.type
    default_name: unknown
    regexes:
      - name: starts_with_letters
        regex: "^[a-zA-Z]+"
      - name: starts_with_numbers
        regex: "^[0-9]+"
```
