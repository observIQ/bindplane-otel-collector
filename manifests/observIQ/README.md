# observIQ manifest

This manifest contains all components available in [OpenTelemetry Collector](https://github.com/open-telemetry/opentelemetry-collector/tree/v0.145.0), [OpenTelemetry Contrib](https://github.com/open-telemetry/opentelemetry-collector-contrib), and custom components defined in this repo.

## Components

This is a list of components that will be available to use in the resulting collector binary.

| extensions                                  | exporters                       | processors                           | receivers                             | connectors               |
| :------------------------------------------ | :------------------------------ | :----------------------------------- | :------------------------------------ | :----------------------- |
| ackextension                                | alertmanagerexporter            | attributesprocessor                  | activedirectorydsreceiver             | countconnector           |
| asapauthextension                           | alibabacloudlogserviceexporter  | batchprocessor                       | aerospikereceiver                     | datadogconnector         |
| avrologencodingextension                    | awscloudwatchlogsexporter       | coralogixprocessor                   | apachereceiver                        | exceptionsconnector      |
| awscloudwatchmetricstreamsencodingextension | awsemfexporter                  | cumulativetodeltaprocessor           | apachesparkreceiver                   | failoverconnector        |
| awslogsencodingextension                    | awskinesisexporter              | datadogsemanticsprocessor            | awscloudwatchreceiver                 | forwardconnector         |
| awsproxy                                    | awss3exporter                   | datapointcountprocessor              | awscontainerinsightreceiver           | grafanacloudconnector    |
| awss3eventextension                         | awsxrayexporter                 | deltatocumulativeprocessor           | awsecscontainermetricsreceiver        | metricsaslogsconnector   |
| azureauthextension                          | azureblobexporter               | deltatorateprocessor                 | awsfirehosereceiver                   | otlpjsonconnector        |
| azureencodingextension                      | azuredataexplorerexporter       | filterprocessor                      | awslambdareceiver                     | roundrobinconnector      |
| badgerextension                             | azureloganalyticsexporter       | geoipprocessor                       | awss3eventreceiver                    | routingconnector         |
| basicauthextension                          | azuremonitorexporter            | groupbyattrsprocessor                | awss3receiver                         | servicegraphconnector    |
| bearertokenauthextension                    | bmchelixexporter                | groupbytraceprocessor                | awss3rehydrationreceiver              | signaltometricsconnector |
| bindplaneextension                          | cassandraexporter               | intervalprocessor                    | awsxrayreceiver                       | slowsqlconnector         |
| cfgardenobserver                            | chronicleexporter               | isolationforestprocessor             | azureblobpollingreceiver              | spanmetricsconnector     |
| cgroupruntimeextension                      | chronicleforwarderexporter      | k8sattributesprocessor               | azureblobreceiver                     | sumconnector             |
| datadogextension                            | clickhouseexporter              | logcountprocessor                    | azureblobrehydrationreceiver          |                          |
| dockerobserver                              | coralogixexporter               | logdedupprocessor                    | azureeventhubreceiver                 |                          |
|                                             | datadogexporter                 | logstransformprocessor               | azuremonitorreceiver                  |                          |
| ecsobserver                                 | datasetexporter                 | lookupprocessor                      | bigipreceiver                         |                          |
| filestorage                                 | debugexporter                   | maskprocessor                        | bindplaneauditlogs                    |                          |
| googleclientauthextension                   | dorisexporter                   | memorylimiterprocessor               | carbonreceiver                        |                          |
| googlecloudlogentryencodingextension        | elasticsearchexporter           | metricextractprocessor               | chronyreceiver                        |                          |
| headerssetterextension                      | faroexporter                    | metricsgenerationprocessor           | ciscoosreceiver                       |                          |
| healthcheckextension                        | fileexporter                    | metricstarttimeprocessor             | cloudflarereceiver                    |                          |
| healthcheckv2extension                      | googlecloudexporter             | metricstatsprocessor                 | cloudfoundryreceiver                  |                          |
| hostobserver                                | googlecloudpubsubexporter       | metricstransformprocessor            | collectdreceiver                      |                          |
| httpforwarderextension                      | googlecloudstorageexporter      | probabilisticsamplerprocessor        | couchdbreceiver                       |                          |
| jaegerencodingextension                     | googlemanagedprometheusexporter | randomfailureprocessor               | datadogreceiver                       |                          |
| jaegerremotesampling                        | honeycombmarkerexporter         | redactionprocessor                   | dockerstatsreceiver                   |                          |
| jsonlogencodingextension                    | influxdbexporter                | remotetapprocessor                   | elasticsearchreceiver                 |                          |
| k8sleaderelector                            | kafkaexporter                   | removeemptyvaluesprocessor           | envoyalsreceiver                      |                          |
| k8sobserver                                 | loadbalancingexporter           | resourceattributetransposerprocessor | expvarreceiver                        |                          |
| kafkatopicsobserver                         | logicmonitorexporter            | resourcedetectionprocessor           | faroreceiver                          |                          |
| oauth2clientauthextension                   | logzioexporter                  | resourceprocessor                    | filelogreceiver                       |                          |
| oidcauthextension                           | mezmoexporter                   | samplingprocessor                    | filestatsreceiver                     |                          |
| opampcustommessages                         | nopexporter                     | schemaprocessor                      | flinkmetricsreceiver                  |                          |
| opampextension                              | opensearchexporter              | snapshotprocessor                    | fluentforwardreceiver                 |                          |
| otlpencodingextension                       | otelarrowexporter               | spancountprocessor                   | githubreceiver                        |                          |
| pebbleextension                             | otlpexporter                    | spanprocessor                        | gitlabreceiver                        |                          |
| pprofextension                              | otlphttpexporter                | sumologicprocessor                   | googlecloudmonitoringreceiver         |                          |
| redisstorageextension                       | prometheusexporter              | tailsamplingprocessor                | googlecloudpubsubpushreceiver         |                          |
| remotetapextension                          | prometheusremotewriteexporter   | throughputmeasurementprocessor       | googlecloudpubsubreceiver             |                          |
| sigv4authextension                          | pulsarexporter                  | topologyprocessor                    | googlecloudspannerreceiver            |                          |
| skywalkingencodingextension                 | qradar                          | transformprocessor                   | googlecloudstoragerehydrationreceiver |                          |
| solarwindsapmsettingsextension              | rabbitmqexporter                | unrollprocessor                      | haproxyreceiver                       |                          |
| sumologicextension                          | sapmexporter                    |                                      | hostmetricsreceiver                   |                          |
| textencodingextension                       | sematextexporter                |                                      | httpcheckreceiver                     |                          |
| zipkinencodingextension                     | sentryexporter                  |                                      | httpreceiver                          |                          |
| zpagesextension                             | signalfxexporter                |                                      | huaweicloudcesreceiver                |                          |
|                                             | snowflakeexporter               |                                      | icmpcheckreceiver                     |                          |
|                                             | splunkhecexporter               |                                      | iisreceiver                           |                          |
|                                             | stefexporter                    |                                      | influxdbreceiver                      |                          |
|                                             | sumologicexporter               |                                      | jaegerreceiver                        |                          |
|                                             | syslogexporter                  |                                      | jmxreceiver                           |                          |
|                                             | tencentcloudlogserviceexporter  |                                      | journaldreceiver                      |                          |
|                                             | tinybirdexporter                |                                      | k8sclusterreceiver                    |                          |
|                                             | webhookexporter                 |                                      | k8seventsreceiver                     |                          |
|                                             | zipkinexporter                  |                                      | k8slogreceiver                        |                          |
|                                             |                                 |                                      | k8sobjectsreceiver                    |                          |
|                                             |                                 |                                      | kafkametricsreceiver                  |                          |
|                                             |                                 |                                      | kafkareceiver                         |                          |
|                                             |                                 |                                      | kubeletstatsreceiver                  |                          |
|                                             |                                 |                                      | libhoneyreceiver                      |                          |
|                                             |                                 |                                      | lokireceiver                          |                          |
|                                             |                                 |                                      | m365receiver                          |                          |
|                                             |                                 |                                      | macosunifiedloggingreceiver           |                          |
|                                             |                                 |                                      | memcachedreceiver                     |                          |
|                                             |                                 |                                      | mongodbatlasreceiver                  |                          |
|                                             |                                 |                                      | mongodbreceiver                       |                          |
|                                             |                                 |                                      | mysqlreceiver                         |                          |
|                                             |                                 |                                      | namedpipereceiver                     |                          |
|                                             |                                 |                                      | netflowreceiver                       |                          |
|                                             |                                 |                                      | nginxreceiver                         |                          |
|                                             |                                 |                                      | nopreceiver                           |                          |
|                                             |                                 |                                      | nsxtreceiver                          |                          |
|                                             |                                 |                                      | ntpreceiver                           |                          |
|                                             |                                 |                                      | oktareceiver                          |                          |
|                                             |                                 |                                      | oracledbreceiver                      |                          |
|                                             |                                 |                                      | osqueryreceiver                       |                          |
|                                             |                                 |                                      | otelarrowreceiver                     |                          |
|                                             |                                 |                                      | otlpjsonfilereceiver                  |                          |
|                                             |                                 |                                      | otlpreceiver                          |                          |
|                                             |                                 |                                      | pcapreceiver                          |                          |
|                                             |                                 |                                      | pluginreceiver                        |                          |
|                                             |                                 |                                      | podmanreceiver                        |                          |
|                                             |                                 |                                      | postgresqlreceiver                    |                          |
|                                             |                                 |                                      | pprofreceiver                         |                          |
|                                             |                                 |                                      | prometheusreceiver                    |                          |
|                                             |                                 |                                      | prometheusremotewritereceiver         |                          |
|                                             |                                 |                                      | pulsarreceiver                        |                          |
|                                             |                                 |                                      | purefareceiver                        |                          |
|                                             |                                 |                                      | purefbreceiver                        |                          |
|                                             |                                 |                                      | rabbitmqreceiver                      |                          |
|                                             |                                 |                                      | receivercreator                       |                          |
|                                             |                                 |                                      | redfishreceiver                       |                          |
|                                             |                                 |                                      | redisreceiver                         |                          |
|                                             |                                 |                                      | restapireceiver                       |                          |
|                                             |                                 |                                      | riakreceiver                          |                          |
|                                             |                                 |                                      | routereceiver                         |                          |
|                                             |                                 |                                      | saphanareceiver                       |                          |
|                                             |                                 |                                      | sapnetweaverreceiver                  |                          |
|                                             |                                 |                                      | signalfxreceiver                      |                          |
|                                             |                                 |                                      | simpleprometheusreceiver              |                          |
|                                             |                                 |                                      | skywalkingreceiver                    |                          |
|                                             |                                 |                                      | snmpreceiver                          |                          |
|                                             |                                 |                                      | snowflakereceiver                     |                          |
|                                             |                                 |                                      | solacereceiver                        |                          |
|                                             |                                 |                                      | splunkenterprisereceiver              |                          |
|                                             |                                 |                                      | splunkhecreceiver                     |                          |
|                                             |                                 |                                      | splunksearchapireceiver               |                          |
|                                             |                                 |                                      | sqlqueryreceiver                      |                          |
|                                             |                                 |                                      | sqlserverreceiver                     |                          |
|                                             |                                 |                                      | sshcheckreceiver                      |                          |
|                                             |                                 |                                      | statsdreceiver                        |                          |
|                                             |                                 |                                      | stefreceiver                          |                          |
|                                             |                                 |                                      | syslogreceiver                        |                          |
|                                             |                                 |                                      | systemdreceiver                       |                          |
|                                             |                                 |                                      | tcpcheckreceiver                      |                          |
|                                             |                                 |                                      | tcplogreceiver                        |                          |
|                                             |                                 |                                      | telemetrygeneratorreceiver            |                          |
|                                             |                                 |                                      | tlscheckreceiver                      |                          |
|                                             |                                 |                                      | udplogreceiver                        |                          |
|                                             |                                 |                                      | vcenterreceiver                       |                          |
|                                             |                                 |                                      | wavefrontreceiver                     |                          |
|                                             |                                 |                                      | webhookeventreceiver                  |                          |
|                                             |                                 |                                      | windowseventlogreceiver               |                          |
|                                             |                                 |                                      | windowseventtracereceiver             |                          |
|                                             |                                 |                                      | windowsperfcountersreceiver           |                          |
|                                             |                                 |                                      | windowsservicereceiver                |                          |
|                                             |                                 |                                      | yanggrpcreceiver                      |                          |
|                                             |                                 |                                      | zipkinreceiver                        |                          |
|                                             |                                 |                                      | zookeeperreceiver                     |                          |
