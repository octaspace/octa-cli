# octa

Command-line interface for the [OctaSpace](https://octa.space) decentralized compute platform.

## Installation

Download the latest binary for your platform from the [releases page](../../releases) and place it in your `PATH`.

## Authentication

Before using any commands, authenticate with your API token:

```bash
octa auth <token>
```

The token is saved to `~/.config/octa/config.yaml` and used automatically for all subsequent commands.

## Account

Show account information and balance:

```bash
octa account
octa account balance
```

## Compute

### List available machines

```bash
octa compute
```

Search by CPU, GPU model, or country:

```bash
octa compute search "rtx 4090"
octa compute search "netherlands"
octa compute search "epyc"
```

### List available applications

```bash
octa compute apps
```

### Deploy a machine

Deploy using an application from the marketplace — the Docker image is resolved automatically:

```bash
octa compute deploy --app <app-uuid> --node <node-id>
```

Deploy with a custom Docker image:

```bash
octa compute deploy --node <node-id> --image ubuntu:22.04
```

Override the image for a marketplace app:

```bash
octa compute deploy --app <app-uuid> --node <node-id> --image myrepo/myimage:latest
```

### Connect to a running session via SSH

```bash
octa compute connect <session-uuid>
```

Supports partial UUIDs. Uses direct SSH by default, falls back to proxy if unavailable.
Force proxy connection:

```bash
octa compute connect <session-uuid> --proxy
```

## Sessions

### List active sessions

```bash
octa sessions
```

### Stop a session

```bash
octa sessions stop <session-uuid>
```

Partial UUIDs are supported — you only need enough characters to uniquely identify the session:

```bash
octa sessions stop abc123
```

## Output formats

Most commands support `-o json` for machine-readable output:

```bash
octa compute -o json
octa compute apps -o json
octa sessions -o json
```
