# observIQ manifest

This manifest contains all components available in [OpenTelemetry Collector](https://github.com/open-telemetry/opentelemetry-collector/tree/v0.105.0), [OpenTelemetryContrib](https://github.com/open-telemetry/opentelemetry-collector-contrib), and custom components defined in this repo. The options available here match parity with what was available to the BindPlane Agent v1.

## Components

This is a list of components that will be available to use in the resulting collector binary.

| extensions                | exporters                      | processors                    | receivers                      | connectors            |
| :------------------------ | :----------------------------- | :---------------------------- | :----------------------------- | :-------------------- |
| zpagesextension           | debugexporter                  | batchprocessor                | nopreceiver                    | forwardconnector      |
| ballastextension          | loggingexporter                | memorylimiterprocessor        | otlpreceiver                   | countconnector        |
| ackextension              | nopexporter                    | attributesprocessor           | activedirectorydsreceiver      | datadogconnector      |
| asapauthextension         | otlpexporter                   | cumulativetodeltaprocessor    | aerospikereceiver              | exceptionsconnector   |
| awsproxy                  | otlphttpexporter               | deltatorateprocessor          | apachereceiver                 | grafanacloudconnector |
| basicauthextension        | alibabacloudlogserviceexporter | filterprocessor               | apachesparkreceiver            | roundrobinconnector   |
| bearertokenauthextension  | awscloudwatchlogsexporter      | groupbyattrsprocessor         | awscloudwatchreceiver          | routingconnector      |
| jaegerencodingextension   | awsemfexporter                 | groupbytraceprocessor         | awscontainerinsightreceiver    | servicegraphconnector |
| otlpencodingextension     | awskinesisexporter             | k8sattributesprocessor        | awsecscontainermetricsreceiver | spanmetricsconnector  |
| zipkinencodingextension   | awsxrayexporter                | metricsgenerationprocessor    | awsfirehosereceiver            |                       |
| headerssetterextension    | awss3exporter                  | metricstransformprocessor     | awsxrayreceiver                |                       |
| healthcheckextension      | azuredataexplorerexporter      | probabilisticsamplerprocessor | azureblobreceiver              |                       |
| httpforwarderextension    | azuremonitorexporter           | redactionprocessor            | azureeventhubreceiver          |                       |
| jaegerremotesampling      | carbonexporter                 | remotetapprocessor            | azuremonitorreceiver           |                       |
| oauth2clientauthextension | cassandraexporter              | resourcedetectionprocessor    | bigipreceiver                  |                       |
| dockerobserver            | clickhouseexporter             | resourceprocessor             | carbonreceiver                 |                       |
| ecsobserver               | coralogixexporter              | routingprocessor              | chronyreceiver                 |                       |
| ecstaskobserver           | datadogexporter                | spanprocessor                 | cloudflarereceiver             |                       |
| hostobserver              | datasetexporter                | sumologicprocessor            | cloudfoundryreceiver           |                       |
| k8sobserver               | elasticsearchexporter          | tailsamplingprocessor         | collectdreceiver               |                       |
| oidcauthextension         | fileexporter                   | transformprocessor            | couchdbreceiver                |                       |
| opampextension            | googlecloudpubsubexporter      | datapointcountprocessor       | datadogreceiver                |                       |
| pprofextension            | honeycombmarkerexporter        | logcountprocessor             | dockerstatsreceiver            |                       |
| sigv4authextension        | influxdbexporter               | logdeduplicationprocessor     | elasticsearchreceiver          |                       |
| filestorage               | instanaexporter                | lookupprocessor               | expvarreceiver                 |                       |
| dbstorage                 | kafkaexporter                  | maskprocessor                 | filelogreceiver                |                       |
| bindplaneextension        | loadbalancingexporter          | metricextractprocessor        | filestatsreceiver              |                       |
|                           | logicmonitorexporter           |
