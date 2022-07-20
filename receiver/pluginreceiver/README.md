# Plugin Receiver

The Plugin Receiver is designed to run templated OpenTelemetry pipelines. This allows users to store complex workflows within a plugin that is loaded by the receiver.

Supported pipeline types: `logs`, `metrics`, `traces`

## Configuration
| Field        | Default | Required | Description |
| ---          | ---     | ---      | ---         |
| `path`       |         | `true`   | The path to the plugin file. |
| `parameters` | { }     | `false`  | A map of `key: value` parameters used to render the plugin's templated pipeline. |

### Example Configuration
```yaml
receivers:
  plugin:
    path: ./plugins/simplehost.yaml
    parameters:
      enable_cpu: false
      enable_memory: true   
```

## Plugins
Plugins are yaml files that define three key aspects:
- Metadata
- Parameters
- Template

### Example
```yaml
title: Simple Host Plugin
description: A plugin utilizing the hostmetrics receiver
version: 0.0.0
parameters:
- name: enable_cpu
  type: bool
  default: false
- name: enable_memory
  type: bool
  default: true
template: |
  receivers:
    hostmetrics:
      scrapers:
  {{if .enable_cpu}}
        cpu:
  {{end}}
  {{if .enable_memory}}
        memory:
  {{end}}
  service:
    pipelines:
      metrics:
        receivers: [hostmetrics]
```
### Metadata
Metadata fields are used to catalog and distinguish plugins. The following fields are required for a plugin:
- `title`
- `description`
- `version`

### Parameters
Parameters are the fields used to configure a plugin. The values of these fields are used when rendering the plugin's template, resulting in a dynamic pipeline. 

The following keys are used when defining a parameter.
| Key | Required | Description |
| --- | --- | --- |
| `name`      | `true`  | The name of the parameter. This is the key used when configuring the parameter within the receiver. |
| `type`      | `true`  | The data type expected for this parameter. Supported values include `string`, `[]string`, `int`, `bool`, `timezone`. |
| `default`   | `false` | The default value of the parameter. If not supplied during configuration, the parameter will default to this value.   |
| `required`  | `false` | Specifies if the parameter must be supplied during configuration. |
| `supported` | `false` | Specifies a list of supported values that can be used for this parameter. |

**Warning**: Parameters must be defined. Undefined parameters will return an error during configuration.

**Warning**: Parameters must adhere to their definition. Invalid parameters will return an error during configuration.

### Template
The plugin template is a templated OpenTelemetry config. When the receiver starts, it uses the plugin's parameters and standard go [templating](https://pkg.go.dev/text/template) to render an internal OpenTelemetry collector.

**Warning**: The supplied template must result in a valid OpenTelemetry config, with the exception of exporter components. Exporters are not supported and should be excluded from the template.

**Warning**: A template can only define one data type. If the template results in two different pipeline data types, such as for logs and metrics, this will result in a configuration error.
