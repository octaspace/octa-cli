# OctaSpace CLI

[![Release](https://img.shields.io/github/v/release/octaspace/octa-cli)](https://github.com/octaspace/octa-cli/releases/latest)
[![Build](https://img.shields.io/github/actions/workflow/status/octaspace/octa-cli/release.yml)](https://github.com/octaspace/octa-cli/actions/workflows/release.yml)

Command-line interface for the [OctaSpace API](https://octa.space). Use it to
inspect account and infrastructure data, deploy machine-rental sessions, manage
sessions, and connect through OctaSpace VPN relays.

The CLI uses the official [OctaSpace Go SDK](https://github.com/octaspace/go-sdk)
for authentication, typed API errors, timeouts, safe-request retries, and API
models. In-flight requests honour `Ctrl-C`.

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

Create an API token in OctaSpace, then authenticate once. The CLI verifies the
token before writing it to the local config file.

```bash
octa auth <token>
octa account
octa compute
octa compute apps
```

The config file is `$XDG_CONFIG_HOME/octa/config.yaml`, or
`~/.config/octa/config.yaml` when `XDG_CONFIG_HOME` is unset. On Unix it is
written with owner-only permissions.

Deploy an application template after choosing a machine and app UUID:

```bash
octa compute deploy --app <app-uuid> --node <node-id>
octa sessions
```

Stop the session when it is no longer needed:

```bash
octa sessions stop <session-uuid-or-unique-prefix>
```

## Commands

| Area | Main commands | Guide |
| --- | --- | --- |
| Authentication and account | `auth`, `account`, `account balance`, `account wallet` | [Getting started](docs/getting-started.md) |
| Nodes | `nodes list`, `show`, `ident`, `logs`, `prices`, `reboot` | [Command reference](docs/commands.md#nodes) |
| Compute | list, search, apps, deploy, logs, SSH | [Compute rentals](docs/compute.md) |
| Sessions | list, info, and stop | [Command reference](docs/commands.md#sessions) |
| Idle jobs | `idle-jobs show`, `idle-jobs logs` | [Command reference](docs/commands.md#idle-jobs) |
| Network | `network` | [Command reference](docs/commands.md#network) |
| VPN | relays, WireGuard, Shadowsocks, OpenVPN, V2Ray | [VPN](docs/vpn.md) |
| Automation | `completion`, `daemon` | [Command reference](docs/commands.md#automation) |

Run `octa <command> --help` for flags and supported values.

## JSON output

Every command has a table and a JSON form (`-o table|json`, table by default).
Most print the server-shaped response through the SDK's resource-scoped raw
methods, keeping fields the API may add before the SDK has a typed model for
them. JSON is an API-server contract, not a versioned CLI schema.

```bash
octa account -o json
octa nodes list -o json
octa nodes show <id> -o json
octa compute -o json
octa compute apps -o json
octa sessions -o json
octa sessions info <uuid> -o json
octa vpn relay list -o json
octa vpn status -o json
```

See [JSON output and compatibility](docs/json-output.md) for the full command
list, including which commands are raw passthrough vs. a typed/derived
payload.

## Documentation

- [Getting started and local configuration](docs/getting-started.md)
- [Compute rentals and deployment defaults](docs/compute.md)
- [VPN relay and tunnel workflow](docs/vpn.md)
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
