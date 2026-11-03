# Command reference

Run `octa <command> --help` for the authoritative flag list installed in the
current binary. Every command takes `-o table|json` unless noted otherwise.

## Account

| Command | Description |
| --- | --- |
| `octa auth <token>` | Verify an API token (by reading the balance) and save it. |
| `octa account` | Show the account balance. |
| `octa account balance` | Same as `octa account`. |

The API has no profile endpoint, so `account` shows the balance only.

## Compute

| Command | Description |
| --- | --- |
| `octa compute` | List machines for rent. Works without a token. |
| `octa compute search <query>` | Search CPU or GPU model, country or city. |
| `octa compute start --node <number> --image <image> [flags]` | Start a compute session (alias: `deploy`); repeat `--node` for a batch. |
| `octa compute start --from-file <batch.json>` | Start a batch of different requests. |
| `octa compute restart <uuid-or-prefix> [--replace flags]` | Recreate the container of a running session. |
| `octa compute logs <uuid-or-prefix> [--tail N]` | Show system events and container output. |
| `octa compute connect <uuid-or-prefix> [--proxy]` | Open an SSH connection (no `-o`). |

See [Compute rentals](compute.md) for the session parameter flags.

## Render

| Command | Description |
| --- | --- |
| `octa render` | List render nodes (NVIDIA GPU with a Blender score). Works without a token. |
| `octa render start --node <number> [--node ...] [--wait]` | Rent render nodes (several `--node` = one batch request), billed hourly until stopped. |

## Sessions

| Command | Description |
| --- | --- |
| `octa sessions` | List running compute and render sessions. |
| `octa sessions info <uuid-or-prefix>` | Show details of one session. |
| `octa sessions stop <uuid-or-prefix>` | Stop a session (asynchronous). |

The API keeps no session history: a stopped session is no longer listed and
its logs are gone.

## Nodes (hosting)

`nodes` has the alias `hosting`.

| Command | Description |
| --- | --- |
| `octa nodes list` | List your online nodes. Nodes that are down are not listed. |
| `octa nodes show <number>` | Show one of your nodes. |
| `octa nodes prices <number> [--base $] [--storage $] [--traffic $] [--min-rent min]` | Update node prices, in USD. |

Prices are given in dollars with up to six decimal places (`--base 0.25`) and
sent as integer micro-USD. `--min-rent` is 0..60 minutes, 0 turns it off.

## Automation

| Command | Description |
| --- | --- |
| `octa completion <bash\|zsh\|fish\|powershell>` | Print a shell-completion script. |
