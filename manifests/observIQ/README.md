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
| basicauthextension        | awss3exporter                  | deltatorateprocessor          | awscloudwatchreceiver          | grafanacloudconnector |
| bearertokenauthextension  | awsxrayexporter                | filterprocessor               | awscontainerinsightreceiver    | roundrobinconnector   |
| bindplaneextension        | azuredataexplorerexporter      | groupbyattrsprocessor         | awsecscontainermetricsreceiver | routingconnector      |
| dbstorage                 | azuremonitorexporter           | groupbytraceprocessor         | awsfirehosereceiver            | servicegraphconnector |
| dockerobserver            | carbonexporter                 | k8sattributesprocessor        | awsxrayreceiver                | spanmetricsconnector  |
| ecsobserver               | cassandraexporter              | logcountprocessor             | azureblobreceiver              |                       |
| ecstaskobserver           | clickhouseexporter             | logdeduplicationprocessor      | azureeventhubreceiver          |                       |
| filestorage               | coralogixexporter              | lookupprocessor               | azuremonitorreceiver           |                       |
| headerssetterextension    | datadogexporter                | maskprocessor                 | bigipreceiver                  |                       |
| healthcheckextension      | datasetexporter                | memorylimiterprocessor        | carbonreceiver                 |                       |
| hostobserver              | debugexporter                  | metricextractprocessor        | chronyreceiver                 |                       |
| httpforwarderextension    | elasticsearchexporter          | metricsgenerationprocessor    | cloudflarereceiver             |                       |
| jaegerencodingextension   | fileexporter                   | metricstransformprocessor     | cloudfoundryreceiver           |                       |
| jaegerremotesampling      | googlecloudpubsubexporter      | probabilisticsamplerprocessor | collectdreceiver               |                       |
| k8sobserver               | honeycombmarkerexporter        | redactionprocessor            | couchdbreceiver                |                       |
| oauth2clientauthextension | influxdbexporter               | remotetapprocessor            | datadogreceiver                |                       |
| oidcauthextension         | instanaexporter                | resourcedetectionprocessor    | dockerstatsreceiver            |                       |
| opampextension            | kafkaexporter                  | resourceprocessor             | elasticsearchreceiver          |                       |
| otlpencodingextension     | loadbalancingexporter          | routingprocessor              | expvarreceiver                 |                       |
| pprofextension            | loggingexporter                | spanprocessor                 | filelogreceiver                |                       |
| sigv4authextension        | logicmonitorexporter           | sumologicprocessor            | filestatsreceiver              |                       |
| zipkinencodingextension   | nopexporter                    | tailsamplingprocessor         | nopreceiver                    |                       |
|                           | otlpexporter                   | transformprocessor            | otlpreceiver                   |                       |
|                           | otlphttpexporter               | unrollprocessor               |                                |                       |
