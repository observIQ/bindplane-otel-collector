# SAP HANA Plugin

Log parser for SAP HANA

## Configuration Parameters

| Name                     | Description                                                                     | Type     | Default                                                                                                                                                                                                       | Required | Values             |
| :----------------------- | :------------------------------------------------------------------------------ | :------- | :------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | :------- | :----------------- |
| file_path                | The directory to the SAP HANA trace logs. No trailing slash should be included. | []string | `[/usr/sap/*/HDB*/*/trace/*.trc]`                                                                                                                                                                             | false    |                    |
| exclude                  | The directories to exclude for the SAP HANA trace logs.                         | []string | `[/usr/sap/*/HDB*/*/trace/nameserver_history*.trc /usr/sap/*/HDB*/*/trace/nameserver*loads*.trc /usr/sap/*/HDB*/*/trace/nameserver*unlaods*.trc /usr/sap/*/HDB*/*/trace/nameserver*executed_statements*.trc]` | false    |                    |
| start_at                 | At startup, where to start reading logs from the file (`beginning` or `end`)    | string   | `end`                                                                                                                                                                                                         | false    | `beginning`, `end` |
| timezone                 | Timezone to use when parsing the timestamp                                      | timezone | `UTC`                                                                                                                                                                                                         | false    |                    |
| offset_storage_dir       | The directory that the offset storage file will be created                      | string   | `${env:OIQ_OTEL_COLLECTOR_HOME}/storage`                                                                                                                                                                      | false    |                    |
| save_log_record_original | Enable to preserve the original log message in a `log.record.original` key.     | string   | `false`                                                                                                                                                                                                       | false    |                    |

## Example Config:

Below is an example of a basic config

```yaml
receivers:
  plugin:
    path: ./plugins/sap_hana_logs.yaml
    parameters:
      file_path: [/usr/sap/*/HDB*/*/trace/*.trc]
      exclude:
        [
          /usr/sap/*/HDB*/*/trace/nameserver_history*.trc /usr/sap/*/HDB*/*/trace/nameserver*loads*.trc /usr/sap/*/HDB*/*/trace/nameserver*unlaods*.trc /usr/sap/*/HDB*/*/trace/nameserver*executed_statements*.trc,
        ]
      start_at: end
      timezone: UTC
      offset_storage_dir: ${env:OIQ_OTEL_COLLECTOR_HOME}/storage
      save_log_record_original: false
```
