# Varnish Cache Receiver

This Varnish Cache Metric Receiver will collects metrics using the [varnishstat](https://varnish-cache.org/docs/7.0/reference/varnishstat.html#varnishstat-1) command. It will generate metrics on the backend, cache, thread and session.

Supported pipeline types: `metrics`

## Prerequisites

This Varnish Cache receiver will collect metrics for [supported versions](https://varnish-cache.org/releases/) 6.X - 7.0.X 

## Configuration

The following configuration settings are optional:

- collection_interval (default = 60s): This receiver collects metrics on an interval. Valid time units are ns, us (or µs), ms, s, m, h.

- working_dir (Optional): The -n flag used in [varnishstat](https://varnish-cache.org/docs/7.0/reference/varnishstat.html#description) to get logs from a varnishd working directory.
- exec_dir (Optional): The directory where the varnishadm and varnishstat executables are located. 

### Example Configuration
```yaml
receivers:
  varnish:
    collection_interval: 60s
```

The full list of settings exposed for this receiver are documented [here](./config.go) with detailed sample configurations [here](./testdata/config.yaml).

## Metrics

Details about the metrics produced by this receiver can be found in [metadata.yaml](./metadata.yaml)
