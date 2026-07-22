# Command reference

Run `octa <command> --help` for the authoritative flag list installed in the
current binary.

## Account

| Command | Description |
| --- | --- |
| `octa auth <token> [-o table\|json]` | Verify and persist an API token. |
| `octa account [-o table\|json]` | Show profile and balance. |
| `octa account balance [-o table\|json]` | Show balance only. |
| `octa account wallet [-o table\|json]` | Generate a new wallet for the account. |

## Nodes

| Command | Description |
| --- | --- |
| `octa nodes list [--country <iso>] [--app <uuid>] [-o table\|json]` | List nodes, optionally filtered. |
| `octa nodes show <id> [-o table\|json]` | Show detailed information about a node. |
| `octa nodes ident <id> [-o <file>]` | Download a node's identity file. |
| `octa nodes logs <id> [-o <file>]` | Download a node's log archive. |
| `octa nodes prices <id> [--base <c>] [--storage <c>] [--traffic <c>] [-o table\|json]` | Update node prices (USD cents). |
| `octa nodes reboot <id> [-o table\|json]` | Reboot a node. |

## Compute

| Command | Description |
| --- | --- |
| `octa compute [-o table\|json]` | List available machine rentals. |
| `octa compute search <query> [-o table\|json]` | Search CPU, GPU, and country fields. |
| `octa compute apps [-o table\|json]` | List application templates. |
| `octa compute deploy --node <id> (--app <uuid>\|--image <image>) [-o table\|json]` | Create a machine-rental session. |
| `octa compute logs <uuid-or-prefix> [-o table\|json]` | Show system and container logs. |
| `octa compute connect <uuid-or-prefix> [--proxy]` | Open an SSH connection. |

## Sessions

| Command | Description |
| --- | --- |
| `octa sessions [-o table\|json]` | List active sessions. |
| `octa sessions info <uuid-or-prefix> [-o table\|json]` | Show details for one matched session. |
| `octa sessions stop <uuid-or-prefix> [-o table\|json]` | Stop one uniquely matched session. |

## Idle jobs

| Command | Description |
| --- | --- |
| `octa idle-jobs show <node-id> <job-id>` | Show the status of an idle job (JSON). |
| `octa idle-jobs logs <node-id> <job-id> [-o <file>]` | Fetch idle-job logs (stdout or file). |

## Network

| Command | Description |
| --- | --- |
| `octa network` | Show network statistics and configuration (JSON). |

## VPN

| Command | Description |
| --- | --- |
| `octa vpn relay list [-o table\|json]` | List VPN relay nodes. |
| `octa vpn relay search <query> [--residential] [-o table\|json]` | Search relays by city or country. |
| `octa vpn relay set <node-id> [-o table\|json]` | Save a relay selection. |
| `octa vpn relay get [-o table\|json]` | Print the saved relay. |
| `octa vpn connect [--protocol ...] [-o table\|json]` | Create a VPN session; WireGuard also creates a local tunnel. |
| `octa vpn disconnect [-o table\|json]` | Remove local WireGuard tunnel and stop recorded session. |
| `octa vpn status [--config\|--qr\|-o table\|json]` | Inspect active VPN configuration. |

## Automation

| Command | Description |
| --- | --- |
| `octa completion <bash\|zsh\|fish\|powershell>` | Print a shell-completion script. |
| `sudo octa daemon` | Run the privileged WireGuard tunnel daemon. |

The daemon serves a local Unix socket and stays in the foreground. It is needed
only for `vpn connect --protocol wg` and `vpn disconnect` local tunnel actions.
