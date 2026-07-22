# Compute rentals

## Browse machines and applications

```bash
octa compute
octa compute search "rtx 4090"
octa compute search "netherlands"
octa compute apps
```

`compute search` matches CPU model, GPU model, and country case-insensitively.
Table output ranks machine offers by GPU count. Add `-o json` to supported list
commands when automation needs API fields beyond the table columns.

## Deploy

Create a session from an application template:

```bash
octa compute deploy --app <app-uuid> --node <node-id>
```

Or provide a custom container image:

```bash
octa compute deploy --node <node-id> --image ubuntu:24.04
```

`--app` always loads the selected template. It inherits its image, minimum disk
size, environment values, ports, HTTP ports, and start command. Explicit CLI
flags override template values; `--envs` overlays only the named variables.

```bash
octa compute deploy --app <app-uuid> --node <node-id> \
  --disk 100 \
  --ports 22,6006 \
  --http-ports 8080,8888 \
  --start-command "python app.py" \
  --entrypoint /bin/sh \
  --envs MODEL=custom,DEBUG=false
```

| Flag | Meaning |
| --- | --- |
| `--app` | Application UUID. Required unless `--image` is supplied. |
| `--node` | Target node ID. Required. |
| `--image` | Container image; overrides the template image. |
| `--disk` | Disk size in GB; overrides template minimum disk size. |
| `--envs` | Comma-separated `KEY=VALUE` pairs; values overlay template defaults. |
| `--ports` | Comma-separated TCP/UDP ports; replaces template ports. |
| `--http-ports` | Comma-separated HTTP ports; replaces template HTTP ports. |
| `--start-command` | Container command; overrides template start command. |
| `--entrypoint` | Container entrypoint. |

The CLI uses the SDK's production-compatible batch create form: one-item array
request with `id: 0`. This is intentionally separate from the SDK's opt-in
`CreateSingle` object form. The CLI does not select that form automatically.

If the API accepts no item, deploy returns the first API rejection reason or
message rather than reporting a blank UUID.

## Logs and SSH

```bash
octa compute logs <session-uuid-or-unique-prefix>
octa compute connect <session-uuid-or-unique-prefix>
octa compute connect <session-uuid-or-unique-prefix> --proxy
```

A partial UUID must match exactly one active session. The SSH command prefers a
direct endpoint and falls back to proxy SSH; `--proxy` skips the direct attempt.
It hands execution to the system `ssh` binary, so that binary must be present on
`PATH`.
