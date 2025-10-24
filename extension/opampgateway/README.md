# OpAMP Gateway

The OpAMP Gateway is an extension that allows the Collector to relay OpAMP messages from
agents to a remote OpAMP server. Often this Collector is also configured as an
[OpenTelemetry Gateway](https://opentelemetry.io/docs/collector/deployment/gateway/) to
receive telemetry from other collectors and forward it to a telemetry backend. Multiple
gateways can be chained together to relay the message from agents through multiple
gateways to the OpAMP server. Like the OpenTelemetry Gateway configuration, the OpAMP
Gateway can be configured on multiple Collectors behind a load balancer.

## Motivations

1. In some deployments, the Collector cannot reach the OpAMP server directly. The
   Collector may be behind a firewall or in a private network and restricted from
   accessing the internet. The OpAMP Gateway allows the Collector to relay messages to the
   remote OpAMP server via a gateway. This is useful in the scenario where the Collector
   can reach the OpAMP Gateway and the OpAMP Gateway can reach the remote OpAMP server.

2. If every agent is required to have its own websocket connection to the OpAMP server,
   scaling to millions of agents becomes more difficult. The OpAMP Gateway can handle
   thousands of agents with a single websocket connection to the OpAMP server. It can also
   be configured to use multiple websocket connections to the OpAMP server to handle even
   more agents.