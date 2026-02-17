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
| azureencodingextension                      | azuredataexplorerexporter       | dnslookupprocessor                   | awslambdareceiver                     | roundrobinconnector      |
| badgerextension                             | azureloganalyticsexporter       | filterprocessor                      | awss3eventreceiver                    | routingconnector         |
| basicauthextension                          | azuremonitorexporter            | geoipprocessor                       | awss3receiver                         | servicegraphconnector    |
| bearertokenauthextension                    | bmchelixexporter                | groupbyattrsprocessor                | awss3rehydrationreceiver              | signaltometricsconnector |
| bindplaneextension                          | cassandraexporter               | groupbytraceprocessor                | awsxrayreceiver                       | slowsqlconnector         |
| cfgardenobserver                            | chronicleexporter               | intervalprocessor                    | azureblobpollingreceiver              | spanmetricsconnector     |
| cgroupruntimeextension                      | chronicleforwarderexporter      | isolationforestprocessor             | azureblobreceiver                     | sumconnector             |
| datadogextension                            | clickhouseexporter              | k8sattributesprocessor               | azureblobrehydrationreceiver          |                          |
| dockerobserver                              | coralogixexporter               | logcountprocessor                    | azureeventhubreceiver                 |                          |
|                                             | datadogexporter                 | logdedupprocessor                    | azuremonitorreceiver                  |                          |
| ecsobserver                                 | datasetexporter                 | logstransformprocessor               | bigipreceiver                         |                          |
| filestorage                                 | debugexporter                   | lookupprocessor                      | bindplaneauditlogs                    |                          |
| googleclientauthextension                   | dorisexporter                   | maskprocessor                        | carbonreceiver                        |                          |
| googlecloudlogentryencodingextension        | elasticsearchexporter           | memorylimiterprocessor               | chronyreceiver                        |                          |
| headerssetterextension                      | faroexporter                    | metricextractprocessor               | ciscoosreceiver                       |                          |
| healthcheckextension                        | fileexporter                    | metricsgenerationprocessor           | cloudflarereceiver                    |                          |
| healthcheckv2extension                      | googlecloudexporter             | metricstarttimeprocessor             | cloudfoundryreceiver                  |                          |
| hostobserver                                | googlecloudpubsubexporter       | metricstatsprocessor                 | collectdreceiver                      |                          |
| httpforwarderextension                      | googlecloudstorageexporter      | metricstransformprocessor            | couchdbreceiver                       |                          |
| jaegerencodingextension                     | googlemanagedprometheusexporter | probabilisticsamplerprocessor        | datadogreceiver                       |                          |
| jaegerremotesampling                        | honeycombmarkerexporter         | randomfailureprocessor               | dockerstatsreceiver                   |                          |
| jsonlogencodingextension                    | influxdbexporter                | redactionprocessor                   | elasticsearchreceiver                 |                          |
| k8sleaderelector                            | kafkaexporter                   | remotetapprocessor                   | envoyalsreceiver                      |                          |
| k8sobserver                                 | loadbalancingexporter           | removeemptyvaluesprocessor           | expvarreceiver                        |                          |
| kafkatopicsobserver                         | logicmonitorexporter            | resourceattributetransposerprocessor | faroreceiver                          |                          |
| oauth2clientauthextension                   | logzioexporter                  | resourcedetectionprocessor           | filelogreceiver                       |                          |
| oidcauthextension                           | mezmoexporter                   | resourceprocessor                    | filestatsreceiver                     |                          |
| opampcustommessages                         | nopexporter                     | samplingprocessor                    | flinkmetricsreceiver                  |                          |
| opampextension                              | opensearchexporter              | schemaprocessor                      | fluentforwardreceiver                 |                          |
| otlpencodingextension                       | otelarrowexporter               | snapshotprocessor                    | githubreceiver                        |                          |
| pebbleextension                             | otlpexporter                    | spancountprocessor                   | gitlabreceiver                        |                          |
| pprofextension                              | otlphttpexporter                | spanprocessor                        | googlecloudmonitoringreceiver         |                          |
| redisstorageextension                       | prometheusexporter              | sumologicprocessor                   | googlecloudpubsubpushreceiver         |                          |
| remotetapextension                          | prometheusremotewriteexporter   | tailsamplingprocessor                | googlecloudpubsubreceiver             |                          |
| sigv4authextension                          | pulsarexporter                  | throughputmeasurementprocessor       | googlecloudspannerreceiver            |                          |
| skywalkingencodingextension                 | qradar                          | topologyprocessor                    | googlecloudstoragerehydrationreceiver |                          |
| solarwindsapmsettingsextension              | rabbitmqexporter                | transformprocessor                   | haproxyreceiver                       |                          |
| sumologicextension                          | sapmexporter                    | unrollprocessor                      | hostmetricsreceiver                   |                          |
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
