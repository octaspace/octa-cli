# Compute rentals

## Browse machines

```bash
octa compute
octa compute search "rtx 4090"
octa compute search "netherlands"
```

`compute search` matches CPU model, GPU model, country and city
case-insensitively. Tables rank machines by GPU count. Machines are identified
by their node number, e.g. `7KQ4-M2XD-4`: case, dashes and spaces do not
matter, `O` reads as `0`, `I` and `L` as `1`.

Prices are in US dollars:

| Column | Meaning |
| --- | --- |
| Price/hr | Base price per hour of rent. |
| Storage, Traffic | Per GB of disk per hour, and per GB of traffic. |
| Min rent | Minimum rent: a session stopped sooner is charged base and storage for the whole minimum. |
| Net ↓/↑ | Measured download/upload speed in Mbit/s; `-` when not measured. |

The listings work without an API token, but the anonymous listing is refreshed
only hourly. With a token it is at most 10 seconds old.

## Start a session

```bash
octa compute start --node <number> --image ubuntu:24.04 --ssh --wait
```

`--node` and `--image` are required. `--wait` follows the session's progress
(image pull, setup) until it is ready, then prints how to reach it.

```bash
octa compute start --node <number> --image python:3.12 \
  --disk 100 \
  --ports 6006 \
  --http-ports 8080,8888 \
  -e MODEL=custom -e DEBUG=false \
  --entrypoint "sh -c" --args "'python app.py'" \
  --startup-script @onstart.sh \
  --ssh --fm --wait
```

| Flag | Meaning |
| --- | --- |
| `--node` | Node number from `octa compute`. Required. |
| `--image` | Container image. Required. |
| `--app` | Application UUID; its settings decide `--ssh` and `--fm` when they are not given. |
| `--disk` | Disk size in GB (API default: 1). |
| `-e`, `--env` | `KEY=VALUE`, repeatable; up to 64 variables, values up to 4 KiB. |
| `--ports` | Comma-separated TCP/UDP ports. 22 is always exposed. |
| `--http-ports` | Comma-separated HTTP ports. 7681 and 7682 are reserved. |
| `--entrypoint` | Replaces the image ENTRYPOINT: one line, split like a shell command, nothing expanded. Use `sh -c` for variables or `&&`. |
| `--args` | Replaces the image CMD: one line. |
| `--startup-script` | Script run as root once the container is up; `@path` reads it from a file (up to 64 KiB). |
| `--ssh` | SSH and a web terminal. `--ssh=false` turns it off; not given = the app's setting (off without an app). If setup fails the session stops. |
| `--fm` | Web file manager **without a login**: anyone with the URL can use it. |
| `--wait`, `--wait-timeout` | Wait until ready (default timeout 15m). |

The `--start-command` and comma-separated `--envs` flags of the legacy CLI are
gone: use `--entrypoint`/`--args` and repeated `-e`. Application templates are
no longer listed by the API, so `--app` only passes the UUID through.

Starting is never retried automatically: a retried start could create a second
paid session.

## Batch start

The API starts several sessions in one request; each request succeeds or fails
on its own. Repeat `--node` to start the same session on several machines:

```bash
octa compute start --node 7KQ4-M2XD-4 --node M95S-WZG8-7 --image ubuntu:24.04 --ssh
```

For requests that differ, pass a JSON array with the API's field names
(`node_number`, `image`, `app`, `disk_size`, `envs`, `ports`, `http_ports`,
`entrypoint`, `args`, `startup_script`, `ssh`, `fm`) and an optional `id`
(string or number, echoed in the result; the position when omitted):

```json
[
  {"id": "train", "node_number": "7KQ4-M2XD-4", "image": "pytorch/pytorch", "disk_size": 100, "ssh": true},
  {"id": "web", "node_number": "M95S-WZG8-7", "image": "nginx", "http_ports": [80]}
]
```

```bash
octa compute start --from-file batch.json --wait
```

`--from-file` cannot be combined with `--node`, `--image`, `--app` or the
parameter flags. Unknown fields are rejected, as the API rejects them. The
command prints one line per request (started with its UUID, or failed with
the API's reason) and exits with an error when any request failed; `-o json`
prints `[{id, node_number, status, uuid|reason}]` (with `session` under
`--wait`). Each start counts toward the API limit of 10 starts a minute.

`render start` takes repeated `--node` the same way.

## Restart

```bash
octa compute restart <uuid-or-prefix>
```

Recreates the container on the same node; the rent, UUID and prices stay.
Without parameter flags the session restarts with the parameters it was started
with.

With any parameter flag (the same as for `start`, except `--node`, `--image`
and `--app`, which a restart cannot change), the given parameters **replace
all** of the session's parameters; they are not merged. Every parameter left
out, or given empty, gets a fixed default rather than the session's value:

| Left out | Becomes |
| --- | --- |
| `--disk` | 1 GB — so `--disk` is required with `--replace` |
| `--entrypoint`, `--args` | the image's own ENTRYPOINT and CMD |
| `--startup-script` | none |
| `-e/--env` | no variables |
| `--ports`, `--http-ports` | no published ports |
| `--ssh`, `--fm` | the app's setting: off for a plain image |

The API does not report the entrypoint, args, startup script or disk size a
session was started with; its env, ports, SSH and file manager are visible only
as the session's current state. So the CLI cannot fill the parameters in: pass
the full set and confirm with `--replace`, or the command refuses to send them.
Before restarting it warns about the env variables, ports, SSH and file manager
the session has now that the new set would drop.

```bash
octa compute restart <uuid-or-prefix> --replace --disk 100 \
  --entrypoint "sh -c" --args "'python app.py'" --ssh -e MODEL=custom --http-ports 8080
```

The API accepts a restart with 202 and applies it asynchronously; if the new
parameters cannot be set up or the command exits, the session stops.

A session can be restarted once in 5 minutes and 6 times an hour; the CLI shows
how long to wait.

## Logs and SSH

```bash
octa compute logs <uuid-or-prefix> --tail 500
octa compute connect <uuid-or-prefix>
octa compute connect <uuid-or-prefix> --proxy
```

`logs` shows the platform events and the latest container output (`--tail`
1..1000 lines, default 200; the output is cut to its last 1 MiB). Logs are
available only while the session runs.

`connect` needs a session started with `--ssh`. It prefers the direct endpoint
and falls back to the proxy; `--proxy` skips the direct one. It logs in as the
user the API reports (the image's USER, root by default) and hands execution to
the system `ssh` binary, which must be on `PATH`.

A partial UUID must match exactly one running session.
