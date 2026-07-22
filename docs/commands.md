# Command reference

Run `octa <command> --help` for the authoritative flag list installed in the
current binary.

## Account

| Command | Description |
| --- | --- |
| `octa auth <token>` | Verify and persist an API token. |
| `octa account [-o table\|json]` | Show profile and balance. |
| `octa account balance [-o table\|json]` | Show balance only. |

## Nodes

| Command | Description |
| --- | --- |
| `octa nodes list [-o table\|json]` | List nodes. |

## Compute

| Command | Description |
| --- | --- |
| `octa compute [-o table\|json]` | List available machine rentals. |
| `octa compute search <query>` | Search CPU, GPU, and country fields. |
| `octa compute apps [-o table\|json]` | List application templates. |
| `octa compute deploy --node <id> (--app <uuid>\|--image <image>)` | Create a machine-rental session. |
| `octa compute logs <uuid-or-prefix>` | Show system and container logs. |
| `octa compute connect <uuid-or-prefix> [--proxy]` | Open an SSH connection. |

## Sessions

| Command | Description |
| --- | --- |
| `octa sessions [-o table\|json]` | List active sessions. |
| `octa sessions stop <uuid-or-prefix>` | Stop one uniquely matched session. |

## VPN

| Command | Description |
| --- | --- |
| `octa vpn relay list [-o table\|json]` | List VPN relay nodes. |
| `octa vpn relay search <query> [--residential]` | Search relays by city or country. |
| `octa vpn relay set <node-id>` | Save a relay selection. |
| `octa vpn relay get` | Print the saved relay. |
| `octa vpn connect [--protocol ...]` | Create a VPN session; WireGuard also creates a local tunnel. |
| `octa vpn disconnect` | Remove local WireGuard tunnel and stop recorded session. |
| `octa vpn status [--config\|--qr\|-o json]` | Inspect active VPN configuration. |

## Automation

| Command | Description |
| --- | --- |
| `octa completion <bash\|zsh\|fish\|powershell>` | Print a shell-completion script. |
| `sudo octa daemon` | Run the privileged WireGuard tunnel daemon. |

The daemon serves a local Unix socket and stays in the foreground. It is needed
only for `vpn connect --protocol wg` and `vpn disconnect` local tunnel actions.
