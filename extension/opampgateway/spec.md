# OpAMP Gateway -- Design Specification

This document describes the internal architecture and design decisions of the
OpAMP Gateway extension. It is aimed at contributors and reviewers.

## Package Layout

```
opampgateway/
  config.go              Config, OpAMPServer, Validate()
  factory.go             NewFactory, default config, configtls -> *tls.Config
  extension.go           Thin OTel extension wrapper (Start/Shutdown)
  doc.go                 Package doc + go:generate
  internal/
    gateway/
      gateway.go         Gateway struct, Settings, New(), callbacks
      server.go          Downstream HTTP/WS server, auth flow
      client.go          Upstream connection manager, pool, assignments
      upstream_connection.go     Single upstream WS connection (reconnect loop)
      downstream_connection.go   Single downstream WS connection
      connection_pool.go         Least-connections selection
      connection_assignments.go  Downstream-to-upstream mapping
      connections.go             Generic thread-safe connection registry
      connection.go              ConnectionCallbacks[T] generic struct
      message.go                 Message wrapper with timestamp
      message_reader.go          WS read loop
      common.go                  WS encode/decode, agent ID parsing
      custom_messages.go         OpampGatewayConnect/Result types, capability
    metadata/              Generated telemetry (mdatagen)
    metadatatest/          Generated telemetry test helpers
```

### Why root vs internal?

The root package is the public surface consumed by the OTel Collector's factory
registry. It exports only `NewFactory()` and `Config`/`OpAMPServer`. All
implementation lives in `internal/gateway/` so it cannot be imported by external
packages. This keeps the public API minimal and allows free refactoring of
internals.

`Config` stays in the root package because `component.Config` must be returned
by `defaultConfig()` in the factory. The internal package defines its own
`Settings` struct to receive the resolved configuration values (including the
`*tls.Config` produced from `configtls.ServerConfig.LoadTLSConfig`). This
avoids a circular import between root and internal.

## Architecture

### Connection Model

```
  N agents  ──ws──>  1 downstream server  ──ws(M)──>  1 upstream OpAMP server
```

The gateway maintains two independent connection layers:

- **Downstream server** (`server.go`): An HTTP server that accepts agent
  WebSocket connections. Each accepted connection becomes a
  `downstreamConnection` with its own reader/writer goroutine pair.

- **Upstream client** (`client.go`): Opens `M` persistent WebSocket connections
  to the upstream OpAMP server at startup. Each connection is an
  `upstreamConnection` with its own reader/writer goroutine pair and automatic
  reconnection with exponential backoff.

### Connection Assignment

When an agent connects, the server asks the client for an upstream connection
via the `UpstreamConnectionAssigner` interface:

```go
type UpstreamConnectionAssigner interface {
    AssignUpstreamConnection(downstreamConnectionID string) (*upstreamConnection, error)
    UnassignUpstreamConnection(downstreamConnectionID string)
}
```

The client implements this interface. Assignment uses **least-connections load
balancing**: `connectionPool.next()` iterates all connected upstream connections
and returns the one with the lowest `downstreamCount`. The assignment is stored
in `connectionAssignments` (a `map[downstreamID]upstreamID`) and persists for
the lifetime of the downstream connection to provide **connection affinity** --
all messages from a given agent always travel over the same upstream connection.

If no upstream connections are currently connected (e.g. during initial startup
or a reconnection), the pool returns `ErrNoUpstreamConnectionsAvailable` and
the server responds with `503 Service Unavailable` and a `Retry-After` header.

### Message Forwarding

Messages are forwarded transparently without inspection beyond what is needed
for routing:

1. **Downstream -> Upstream**: The `HandleDownstreamMessage` callback decodes the
   `AgentToServer` protobuf to extract `InstanceUid` (the agent ID). It registers the
   agent-to-downstream mapping for reply routing, then forwards the raw message bytes to
   the assigned upstream connection's write channel.

2. **Upstream -> Downstream**: The `HandleUpstreamMessage` callback decodes the
   `ServerToAgent` protobuf to extract `InstanceUid`, looks up the downstream connection
   for that agent, and forwards the raw message bytes to that connection's write channel.

Messages are not re-encoded. The gateway preserves the original protobuf bytes
end-to-end.

### Reader/Writer Goroutine Pattern

Both `upstreamConnection` and `downstreamConnection` use the same pattern:

- A **reader goroutine** calls `conn.ReadMessage()` in a loop. On error or
  context cancellation it returns, which cancels the writer's context.
- A **writer goroutine** selects on the write channel and the context. When
  the context is cancelled, it closes the WebSocket connection (which unblocks
  `ReadMessage` in the reader).
- The `start()` method blocks until both goroutines finish, then calls
  `OnClose`.

The write channel is unbuffered. This provides natural backpressure: if the
writer is slow, the sender blocks. The read error channel is buffered (size 1)
so the reader goroutine never blocks when reporting an error.

### Upstream Reconnection

Each `upstreamConnection` has an outer loop in `startWriter` that calls
`ensureConnected` (infinite exponential backoff) followed by `writerLoop`.
When `writerLoop` returns (connection lost), the loop iterates: the reader is
cancelled and awaited, `OnClose` is called, and `ensureConnected` retries.

A `nextMessage` variable preserves the last undelivered message across
reconnections, so a message that failed to write is retried on the new
connection rather than lost.

## Authentication

Agent authentication is delegated to the upstream OpAMP server. The flow:

1. Agent HTTP request arrives at the downstream server.
2. An upstream connection is assigned.
3. The server constructs an `OpampGatewayConnect` JSON payload containing:
   - `request_uid`: a UUID for correlation
   - `remote_address`: the agent's socket address
   - `headers`: the agent's HTTP headers (including `Authorization`)
4. This payload is wrapped in an `AgentToServer` protobuf `CustomMessage`
   (capability: `com.bindplane.opamp-gateway`, type: `connect`) and sent
   upstream.
5. The server blocks (with a 30-second timeout) waiting on a per-request
   channel in `pendingAuthRequests`.
6. When the upstream responds with an `OpampGatewayConnectResult` custom
   message (type: `connectResult`), `handleAuthResponse` routes it to the
   correct channel using the `request_uid`.
7. If accepted, the WebSocket upgrade proceeds. If rejected, the agent
   receives the HTTP status code and headers from the result.

This design means the upstream server has full control over authentication
policy. The gateway itself never inspects or validates credentials.

### Why authenticate before the WebSocket upgrade?

Authenticating before `Upgrade()` allows the gateway to return proper HTTP
error codes (401, 403, etc.) to the agent. Once a WebSocket is upgraded, the
only way to reject is to close the connection, which provides no structured
error information to the client.

## Lifecycle (Start/Shutdown/Restart)

The gateway supports being started, shut down, and restarted on the same
instance. This happens during OTel Collector hot-reloads.

### Start

1. `client.Start()` resets all internal state (pool, connections, assignments,
   WaitGroup), derives a cancellable context, and launches upstream connection
   goroutines.
2. `server.Start()` resets internal state (shutdown context, connection maps,
   pending auth requests), creates a new `http.Server`, binds a TCP listener,
   and serves in the background.

### Shutdown

1. `client.Stop()` cancels the client context and waits for all upstream
   connection goroutines to finish. Each goroutine's deferred cleanup removes
   itself from the pool and connection registry.
2. `server.Stop()` cancels the shutdown context (which stops all downstream
   connection goroutines), shuts down the HTTP server with a 10-second timeout,
   and waits for all downstream connection goroutines to finish.

### Restart Safety

Both `Start` methods reset all mutable state before creating new goroutines.
This is critical because:

- `server.shutdownCtx` is permanently cancelled after `Stop()`. Without a
  reset, all new downstream connections would receive a cancelled context.
- Connection maps and pending auth requests could contain stale entries from the
  previous lifecycle.
- The client's pool and assignments must be empty so new upstream connections
  start with correct counts.

`Shutdown` is safe to call without a prior `Start`, and safe to call multiple
times (context cancel and WaitGroup wait are both idempotent).

## Telemetry

Four metrics are emitted, each tagged with `direction=upstream|downstream`:

| Metric | Type | What it measures |
|--------|------|-----------------|
| `opampgateway.connections` | UpDown Sum | Current number of open connections |
| `opampgateway.messages` | Monotonic Sum | Total messages forwarded |
| `opampgateway.message.bytes` | Monotonic Sum | Total bytes forwarded |
| `opampgateway.messages.latency` | Histogram (ms) | Time from message receipt to forwarding |

Connection counts are incremented when a connection starts and decremented in a
deferred cleanup, ensuring accuracy even on error paths.

Latency is measured from when `newMessage()` is called (message receipt) to when
`writeWSMessage()` completes. This captures time spent in the write channel
(queuing) plus serialization, but not network transit.

## Concurrency

- **Connection registries** (`connections[T]`): Protected by `sync.RWMutex`.
  Reads (get, size) use `RLock`; writes (set, remove) use full `Lock`.

- **Connection pool**: Same RWMutex pattern. `next()` (read-only iteration)
  uses `RLock`.

- **Connection assignments**: Protected by `sync.Mutex` (all operations mutate
  the map or read-then-write).

- **Pending auth requests**: Protected by `sync.Mutex`. Each request gets a
  buffered channel so the sender (upstream message handler) never blocks.

- **Upstream connection state**: `connected` and `count` use `atomic.Bool` and
  `atomic.Int32` respectively, avoiding locks on the hot path.

- **Write channels**: Unbuffered channels serialize writes per connection.
  Only the writer goroutine calls `writeWSMessage`; no locking is needed on
  the WebSocket connection for writes.

## Wire Format

The gateway speaks the OpAMP WebSocket wire format:

- Messages are binary WebSocket frames.
- Each frame optionally starts with a zero-byte varint header (per the OpAMP
  spec grace period). The decoder handles both old (no header) and new (header
  present) formats.
- The protobuf payload is `AgentToServer` (upstream) or `ServerToAgent`
  (downstream) from `opamp-go/protobufs`.
- Custom messages use the `CustomMessage` field with capability
  `com.bindplane.opamp-gateway`.

## Key Dependencies

| Dependency | Purpose |
|-----------|---------|
| `gorilla/websocket` | WebSocket client and server |
| `opamp-go/protobufs` | OpAMP protobuf message types |
| `cenkalti/backoff/v4` | Exponential backoff for upstream reconnection |
| `json-iterator/go` | JSON marshal/unmarshal for custom messages |
| `google/uuid` | Agent ID parsing and auth request correlation |
| `configtls` | OTel-native TLS configuration deserialization |
