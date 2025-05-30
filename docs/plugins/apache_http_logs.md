# Apache HTTP Server Plugin

Log parser for Apache HTTP Server
For optimal Apache HTTP parsing and enrichment, we recommend choosing 'observIQ' log format in which log Apache logging configuration is modified to output log entries as JSON files.

Steps for updating config file apache2.conf:

1. Add the access Logformat and error ErrorLogFormat to the main apache configuration.
   On Debian based systems, this can be found in /etc/apache2/apache2.conf
2. Modify CustomLog in sites-available configurations to use 'observiq' for the access log format.
   ex: CustomLog ${env:APACHE_LOG_DIR}/access.log observiq
3. Restart Apache Http Server

The 'observIQ' log format is defined for access logs and error logs as follows:
Logformat "{\"timestamp\":\"%{%Y-%m-%dT%T}t.%{usec_frac}t%{%z}t\",\"remote_addr\":\"%a\",\"protocol\":\"%H\",\"method\":\"%m\",\"query\":\"%q\",\"path\":\"%U\",\"status\":\"%>s\",\"http_user_agent\":\"%{User-agent}i\",\"http_referer\":\"%{Referer}i\",\"remote_user\":\"%u\",\"body_bytes_sent\":\"%b\",\"request_time_microseconds\":\"%D\",\"http_x_forwarded_for\":\"%{X-Forwarded-For}i\"}" observiq
ErrorLogFormat "{\"time\":\"%{cu}t\",\"module\":\"%-m\",\"client\":\"%-a\",\"http_x_forwarded_for\":\"%-{X-Forwarded-For}i\",\"log_level\":\"%-l\",\"pid\":\"%-P\",\"tid\":\"%-T\",\"message\":\"%-M\",\"logid\":{\"request\":\"%-L\",\"connection\":\"%-{c}L\"},\"request_note_name\":\"%-{name}n\"}"

## Configuration Parameters

| Name                     | Description                                                                                                                                                                                                                                                                                                                          | Type     | Default                         | Required | Values                |
| :----------------------- | :----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | :------- | :------------------------------ | :------- | :-------------------- |
| log_format               | When choosing the 'default' option, the agent will expect and parse logs in a format that matches the default logging configuration. When choosing the 'observIQ' option, the agent will expect and parse logs in an optimized JSON format that adheres to the observIQ specification, requiring an update to the apache2.conf file. | string   | `default`                       | false    | `default`, `observiq` |
| enable_access_log        | Enable to collect Apache HTTP Server access logs                                                                                                                                                                                                                                                                                     | bool     | `true`                          | false    |                       |
| access_log_path          | Path to access log file                                                                                                                                                                                                                                                                                                              | []string | `[/var/log/apache2/access.log]` | false    |                       |
| enable_error_log         | Enable to collect Apache HTTP Server error logs                                                                                                                                                                                                                                                                                      | bool     | `true`                          | false    |                       |
| error_log_path           | Path to error log file                                                                                                                                                                                                                                                                                                               | []string | `[/var/log/apache2/error.log]`  | false    |                       |
| start_at                 | At startup, where to start reading logs from the file (`beginning` or `end`)                                                                                                                                                                                                                                                         | string   | `end`                           | false    | `beginning`, `end`    |
| timezone                 | Timezone to use when parsing the timestamp                                                                                                                                                                                                                                                                                           | timezone | `UTC`                           | false    |                       |
| save_log_record_original | Enable to preserve the original log message in a `log.record.original` key.                                                                                                                                                                                                                                                          | string   | `false`                         | false    |                       |

## Example Config:

Below is an example of a basic config

```yaml
receivers:
  plugin:
    path: ./plugins/apache_http_logs.yaml
    parameters:
      log_format: default
      enable_access_log: true
      access_log_path: [/var/log/apache2/access.log]
      enable_error_log: true
      error_log_path: [/var/log/apache2/error.log]
      start_at: end
      timezone: UTC
      save_log_record_original: false
```
