# JSON output and API compatibility

Commands that read one API resource pass the SDK's raw response through to
stdout, unmodified apart from the CLI's trailing newline:

```bash
octa compute -o json
octa render -o json
octa sessions -o json
octa sessions info <uuid> -o json
octa nodes list -o json
```

These keep every API field, including ones the SDK's typed models do not have
yet. They do not decode and re-marshal the payload.

The other read commands marshal the SDK's typed model: they filter client-side
or the SDK has no raw accessor for the endpoint.

```bash
octa account -o json            # {"balance", "currency", "decimals"}
octa compute search <query> -o json
octa compute logs <uuid> -o json
octa nodes show <number> -o json
```

Typed re-marshaling can drop fields the API added since the model was last
updated. Prefer a raw command when a script needs the complete server response.

Commands that change something (`auth`, `compute start`, `compute restart`,
`render start`, `sessions stop`, `nodes prices`) print a small confirmation
object such as `{"status": "ok", "uuid": "..."}`. With `--wait`, `compute
start`, `compute restart` and `render start` print the ready session instead.
A batch start prints `[{"id", "node_number", "status", "uuid" | "reason"}]`
(`status` 0 = started, 1 = failed), with `session` under `--wait`.

## Units

| Field | Unit |
| --- | --- |
| `*_price`, `charge_amount`, `prices.*`, `balance` | integer micro-USD (1 = $0.000001); for `balance` use `decimals` |
| `started_at` (sessions), `system[].ts` (logs) | Unix milliseconds |
| `duration`, `uptime` | seconds |
| `online_at` (nodes) | ISO 8601 |
| `net_down_mbps`, `net_up_mbps` | Mbit/s; 0 = not measured |
| `memory`, `disk` (listings) | free bytes; `node_hw.disk` of a session is the rented GB |

The JSON shape of raw-passthrough commands is controlled by the OctaSpace API.
Scripts should tolerate unknown fields
and validate the fields they consume.

Table output is different by design: it uses typed SDK models and selects a
concise set of human-readable fields.

The API is described by its OpenAPI specification at
`https://api-dev.octa.space/openapi.json` (Swagger UI: `/swagger`). For SDK
models and transport behaviour, see the
[Go SDK documentation](https://github.com/octaspace/go-sdk).
