# observIQ manifest

This manifest contains all components available in [OpenTelemetry Collector](https://github.com/open-telemetry/opentelemetry-collector/tree/v0.105.0), [OpenTelemetryContrib](https://github.com/open-telemetry/opentelemetry-collector-contrib), and custom components defined in this repo. The options available here match parity with what was available to the BDOT v1.

## Components

This is a list of components that will be available to use in the resulting collector binary.

| extensions                | exporters                      | processors                    | receivers                      | connectors            |
| :------------------------ | :----------------------------- | :---------------------------- | :----------------------------- | :-------------------- |
| ackextension              | alibabacloudlogserviceexporter | attributesprocessor           | activedirectorydsreceiver      | countconnector        |
| asapauthextension         | awscloudwatchlogsexporter      | batchprocessor                | aerospikereceiver              | datadogconnector      |
| awsproxy                  | awsemfexporter                 | cumulativetodeltaprocessor    | apachereceiver                 | exceptionsconnector   |
| ballastextension          | awskinesisexporter             | datapointcountprocessor       | apachesparkreceiver            | forwardconnector      |
| basicauthextension        | awss3exporter                  | deltatocumulativeprocessor    | awscloudwatchreceiver          | grafanacloudconnector |
| bearertokenauthextension  | awsxrayexporter                | deltatorateprocessor          | awscontainerinsightreceiver    | roundrobinconnector   |
| bindplaneextension        | azuredataexplorerexporter      | filterprocessor               | awsecscontainermetricsreceiver | routingconnector      |
| dbstorage                 | azuremonitorexporter           | groupbyattrsprocessor         | awsfirehosereceiver            | servicegraphconnector |
| dockerobserver            | carbonexporter                 | groupbytraceprocessor         | awsxrayreceiver                | spanmetricsconnector  |
| ecsobserver               | cassandraexporter              | k8sattributesprocessor        | azureblobreceiver              |                       |
| ecstaskobserver           | clickhouseexporter             | logcountprocessor             | azureeventhubreceiver          |                       |
| filestorage               | coralogixexporter              | logdeduplicationprocessor     | azuremonitorreceiver           |                       |
| headerssetterextension    | datadogexporter                | lookupprocessor               | bigipreceiver                  |                       |
| healthcheckextension      | datasetexporter                | maskprocessor                 | carbonreceiver                 |                       |
| hostobserver              | debugexporter                  | memorylimiterprocessor        | chronyreceiver                 |                       |
| httpforwarderextension    | elasticsearchexporter          | metricextractprocessor        | cloudflarereceiver             |                       |
| jaegerencodingextension   | fileexporter                   | metricsgenerationprocessor    | cloudfoundryreceiver           |                       |
| jaegerremotesampling      | googlecloudpubsubexporter      | metricstransformprocessor     | collectdreceiver               |                       |
| k8sobserver               | honeycombmarkerexporter        | probabilisticsamplerprocessor | couchdbreceiver                |                       |
| oauth2clientauthextension | influxdbexporter               | redactionprocessor            | datadogreceiver                |                       |
| oidcauthextension         | instanaexporter                | remotetapprocessor            | dockerstatsreceiver            |                       |
| opampextension            | kafkaexporter                  | resourcedetectionprocessor    | elasticsearchreceiver          |                       |
| otlpencodingextension     | loadbalancingexporter          | resourceprocessor             | expvarreceiver                 |                       |
| pprofextension            | loggingexporter                | routingprocessor              | filelogreceiver                |                       |
| sigv4authextension        | logicmonitorexporter           | spanprocessor                 | filestatsreceiver              |                       |
| zipkinencodingextension   | nopexporter                    | sumologicprocessor            | githubreceiver                 |                       |
|                           | otlpexporter                   | tailsamplingprocessor         | nopreceiver                    |                       |
|                           | otlphttpexporter               | transformprocessor            | otlpreceiver                   |                       |
|                           |                                | unrollprocessor               |                                |                       |
