# OpAMP Configuration

The Bindplane Agent can be setup as an agent that is managed by the [Bindplane OP platform](https://observiq.com/) via OpAMP.

## Configuration

The agent can be configured to connect to the server a few different ways.

### Config File

The agent can be configured to read its connection config from a `manager.yaml` file. The `--manager` flag can be used to specify the location of this config file, by default it's `./manager.yaml`. The contents of the `manager.yaml` are detailed out in the table below.

| Parameter  | Required | Description                                                                |
| :--------  | :------: | :------------------------------------------------------------------------- |
| endpoint   | X        | The API endpoint to communicate with the server via websocket              |
| secret_key |          | The Secret Key defined for the server to be used for authorization         |
| agent_id   |          | A [ULID](https://github.com/ulid/spec) used to uniquely identify the agent |
| labels     |          | A comma separated list of labels in the form `label=value`                 |
| agent_name |          | Human readable name for the agent                                          |
| tls_config |          | See [tls config](#tls-config) section                                      |

Here's an example of what a common `manager.yaml` looks like:

```yaml
endpoint: ws://bindplane.localnet/v1/opamp
secret_key: 3d83f0cb-2567-42c7-ada6-960842924d11
agent_id: 01H5MG7N9N36J28WEC8A8X5B17
```

#### TLS Config

If TLS is enabled on the server the agent will need to be configured in order to connect. 

**Note**: If using TLS on the server the `endpoint` field will need to have the `wss` protocol for TLS enabled websockets.

| Parameter            | Required | Description                                                                                         |
| :------------------- | :------: | :-------------------------------------------------------------------------------------------------- |
| insecure_skip_verify |          | InsecureSkipVerify controls whether a client verifies the server's certificate chain and host name. |
| key_file             |          | Path to the `.key` file                                                                             |
| cert_file            |          | Path to the Certificate file                                                                        |
| ca_file              |          | Path to the Certificate Authority file                                                              |

### Environment variables

The agent can also use environment variables to set portions of the connection configuration. This is useful for a containerized agent where a mounted volume might not be present. 

If the agent can not find the specified `manager.yaml` file it will search for the environment variables and create a `manager.yaml` at the location of the `--manager` command argument.

**Note**: Only the `OPAMP_ENDPOINT` is required. If this is not set and there is no `manager.yaml` the agent will start in its normal standalone mode.

| Environment Variable  | Required | Description                                                                       |
| :-------------------- | :------: | :-------------------------------------------------------------------------------- |
| OPAMP_ENDPOINT        | X        | The API endpoint to communicate with the server via websocket                     |
| OPAMP_SECRET_KEY      |          | The Secret Key defined for the server to be used for authorization                |
| OPAMP_AGENT_ID        |          | A UUID used to uniquely identify the agent. If not supplied one will be generated |
| OPAMP_LABELS          |          | A comma separated list of labels in the form `label=value`                        |
| OPAMP_AGENT_NAME      |          | Human readable name for the agent                                                 |
| OPAMP_TLS_SKIP_VERIFY |          | Set to `"true"` to skip verification of the OpAMP server's TLS certificate        |
| OPAMP_TLS_CA          |          | File path to a certificate authority file that should be used to validate the server's TLS certificate |
| OPAMP_TLS_CERT        |          | File path to a certificate file that will be used for client TLS authentication |
| OPAMP_TLS_KEY         |          | File path to a private key file that will be used for client TLS authentication |
