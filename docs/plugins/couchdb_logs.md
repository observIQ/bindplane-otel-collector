# CouchDB Plugin

Log parser for CouchDB

## Configuration Parameters

| Name                     | Description                                                                  | Type     | Default                          | Required | Values             |
| :----------------------- | :--------------------------------------------------------------------------- | :------- | :------------------------------- | :------- | :----------------- |
| log_paths                | The absolute path to the CouchDB logs                                        | []string | `[/var/log/couchdb/couchdb.log]` | false    |                    |
| start_at                 | At startup, where to start reading logs from the file (`beginning` or `end`) | string   | `end`                            | false    | `beginning`, `end` |
| save_log_record_original | Enable to preserve the original log message in a `log.record.original` key.  | string   | `false`                          | false    |                    |

## Example Config:

Below is an example of a basic config

```yaml
receivers:
  plugin:
    path: ./plugins/couchdb_logs.yaml
    parameters:
      log_paths: [/var/log/couchdb/couchdb.log]
      start_at: end
      save_log_record_original: false
```
