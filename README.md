# OctaSpace CLI

[![Release](https://img.shields.io/github/v/release/octaspace/octa-cli)](https://github.com/octaspace/octa-cli/releases/latest)
[![Build](https://img.shields.io/github/actions/workflow/status/octaspace/octa-cli/release.yml)](https://github.com/octaspace/octa-cli/actions/workflows/release.yml)

Command-line interface for the [OctaSpace API](https://octa.space). Use it to
rent machines and render nodes, manage running sessions, and manage the prices
of the nodes you host.

> **Branch `next`.** This branch targets the next-generation OctaSpace API,
> which is in active development. Its default host is the test stand
> `https://api-dev.octa.space`; the production hosts still serve the legacy
> API, which `main` supports. Set `OCTA_API_URL` to use another host.

The CLI uses the official [OctaSpace Go SDK](https://github.com/octaspace/go-sdk)
(its `next` branch) for authentication, typed API errors, timeouts,
safe-request retries, and API models. In-flight requests honour `Ctrl-C`.

## Requirements and installation

Download a platform binary from the [release page](https://github.com/octaspace/octa-cli/releases), put it on your `PATH`, and verify it:

```bash
octa --version
```

To build from source, Go 1.25 or later is required:

```bash
git clone https://github.com/octaspace/octa-cli.git
cd octa-cli
make build
./octa --version
```

## Quick start

Create an API key in OctaSpace (Profile → Credentials, it starts with
`octa_ak_`), then authenticate once. The CLI verifies the token before writing
it to the local config file.

```bash
octa auth <token>
octa account
octa compute
```

The config file is `$XDG_CONFIG_HOME/octa/next.yaml`, or
`~/.config/octa/next.yaml` when `XDG_CONFIG_HOME` is unset. On Unix it is
written with owner-only permissions. It is separate from the `config.yaml` of
the legacy-API CLI, whose token this branch never reads.

Start a session on a machine from the listing and wait until it is ready:

```bash
octa compute start --node <node-number> --image ubuntu:24.04 --ssh --wait
octa compute connect <session-uuid-or-unique-prefix>
```

Stop the session when it is no longer needed:

```bash
octa sessions stop <session-uuid-or-unique-prefix>
```

## Commands

| Area | Main commands | Guide |
| --- | --- | --- |
| Authentication and account | `auth`, `account` | [Getting started](docs/getting-started.md) |
| Compute | list, search, start, restart, logs, SSH | [Compute rentals](docs/compute.md) |
| Render | `render`, `render start` | [Command reference](docs/commands.md#render) |
| Sessions | list, info, stop | [Command reference](docs/commands.md#sessions) |
| Hosting | `nodes list`, `show`, `prices` | [Command reference](docs/commands.md#nodes-hosting) |
| Automation | `completion` | [Command reference](docs/commands.md#automation) |

Run `octa <command> --help` for flags and supported values.

## JSON output

Every command has a table and a JSON form (`-o table|json`, table by default).
Most print the server-shaped response through the SDK's resource-scoped raw
methods, keeping fields the API may add before the SDK has a typed model for
them. JSON is an API-server contract, not a versioned CLI schema.

```bash
octa account -o json
octa compute -o json
octa render -o json
octa sessions -o json
octa sessions info <uuid> -o json
octa nodes list -o json
```

See [JSON output and compatibility](docs/json-output.md) for the full command
list, including which commands are raw passthrough vs. a typed/derived
payload.

## Documentation

- [Getting started and local configuration](docs/getting-started.md)
- [Compute rentals](docs/compute.md)
- [Command reference](docs/commands.md)
- [JSON output and API compatibility](docs/json-output.md)
- [Development and verification](docs/development.md)

## Run and test

```bash
make build
go test ./...
go test -race ./...
go vet ./...
gofmt -l .
```

Tests are offline and do not need an API token. More detail: [development and
verification](docs/development.md).
