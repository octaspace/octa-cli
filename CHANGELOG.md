# Changelog

All notable changes to this project will be documented in this file.

The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).
This project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

Changelog tracking starts with `0.0.8`; earlier tags (`v0.0.1`–`v0.0.7`) are
recorded only through their Git history and GitHub releases.

---

## [Unreleased]

The CLI moves to the next-generation OctaSpace API (branch `next`), through
the `next` branch of `github.com/octaspace/go-sdk`. The new API is not
compatible with the legacy one: this version does not work against the legacy
production hosts.

### Changed

- **Breaking:** the default API host is the test stand of the new API,
  `https://api-dev.octa.space`, until its production host is announced.
  `OCTA_API_URL` or `api_url` in the config file select another host.
- **Breaking:** API keys are sent as `Authorization: Bearer <key>`. The new API
  accepts the `octa_ak_…` keys issued in Profile → Credentials; tokens of the
  legacy API are rejected. `auth` verifies the token by reading the balance.
- **Breaking:** machines and nodes are identified by their node number (e.g.
  `7KQ4-M2XD-4`) instead of an integer ID, in every command.
- **Breaking:** prices, charges and the balance are shown in US dollars from
  the API's integer micro-USD; OCTA/Wei amounts are gone.
- **Breaking:** `account` shows the balance only: the API has no profile
  endpoint any more.
- **Breaking:** `compute deploy` becomes `compute start` (`deploy` stays an
  alias): `--node` takes a node number, `--image` is required, `--envs` is
  replaced by repeatable `-e/--env KEY=VALUE`, and `--start-command` by
  `--entrypoint` + `--args`. Application templates are no longer resolved.
- **Breaking:** `nodes list` / `nodes show` read the account's hosted nodes
  (`/hosting/nodes`, online nodes only); `nodes list` loses `--country` and
  `--app`. `nodes prices` takes the node number and prices in dollars, and
  gains `--min-rent`.
- **Breaking:** the config file is `~/.config/octa/next.yaml`, separate from
  the legacy CLI's `config.yaml`; run `octa auth <key>` once. Keys that do not
  start with `octa_ak_` are rejected without being sent.
- `compute` and `render` listings work without a token (refreshed hourly).
- Tables show the minimum rent, disk type and measured network speed
  (Mbit/s); `sessions` shows the service, the start progress and the SSH user.
- API errors show the API's own message; 409 and 503 are handled, and rate
  limits show the `Retry-After` wait.

### Added

- `compute start` flags `--args`, `--startup-script` (`@file`), `--ssh`,
  `--fm` (tri-state: not given = the app's setting) and `--wait` /
  `--wait-timeout`, with client-side checks of the API limits.
- `compute restart <uuid>`: recreate the container with the original
  parameters, or with a full new set confirmed by `--replace`. Parameter flags
  replace all of the session's parameters with fixed defaults for the rest, so
  they are refused without `--replace`, `--disk` is required with it (a
  restart without disk_size shrinks the disk to 1 GB), and the CLI warns about
  the env, ports, SSH and file manager the session would lose.
- Batch starts: repeat `--node` in `compute start` / `render start`, or pass
  `compute start --from-file batch.json` (a JSON array in the API's field
  names). Each request succeeds or fails on its own; the command reports every
  result and fails when any request failed.
- `compute logs --tail N`; truncated or unavailable container output is marked.
- `render` and `render start --node <number>`.
- `sessions stop` warns when a session is stopped before its minimum rent.
- `nodes` alias `hosting`.
- `compute start` and `render start` reject stray positional arguments (an
  unquoted value with spaces) instead of silently dropping them.

### Removed

- **Breaking:** commands without a counterpart in the new public API:
  `account wallet`, `compute apps`, `nodes ident`, `nodes logs`,
  `nodes reboot`, `idle-jobs`, `vpn` (relays, WireGuard, Shadowsocks, OpenVPN,
  V2Ray) and the WireGuard `daemon`, with the WireGuard and QR-code
  dependencies.
- `network`: the public statistics endpoints are not part of the CLI.

---

## [0.2.0] - 2026-07-22

### Added

- `-o table|json` on every command that was still missing it: `compute
  search`, `compute deploy`, `compute logs`, `nodes show`, `nodes prices`,
  `nodes reboot`, `sessions stop`, `vpn relay search`, `vpn relay set`,
  `vpn relay get`, `vpn connect`, `vpn disconnect`, `auth`. Endpoints without
  a raw SDK accessor (`nodes show`, `compute search`, `compute logs`,
  `vpn relay search`) marshal the typed model instead of a server passthrough
  — see `docs/json-output.md` for which commands fall into each category.

### Fixed

- Bumped `github.com/octaspace/go-sdk` to `v0.5.0`: `container_id` comes back
  as a JSON array for compute (MR) sessions, which broke typed decoding of
  the entire `/sessions` list and `/services/:uuid/info` for any account with
  an active compute rental. That took down `octa sessions`, `sessions
  stop`/`info`, `compute logs`/`connect`, and `vpn status` — not just the one
  field. Verified against a live account with an active session.
- Argument-count errors (e.g. `octa nodes prices` with no ID) now name the
  missing argument and print the command's usage line instead of cobra's bare
  `accepts 1 arg(s), received 0`.
- `octa nodes show` no longer prints a fabricated `Prices — (see 'nodes
  list')` row; the `/nodes/:id` endpoint doesn't return prices, so the row is
  simply omitted.

### Changed

- Removed the per-state/per-row color palette from table and label/value
  output. Headers stay bold; everything else renders in the terminal's
  default foreground color, for readability across terminal themes.

---

## [0.1.1] - 2026-07-22

### Fixed

- `octa nodes show <id>` now reports CPU, RAM, disk, GPU, uptime and agent
  version correctly. The `/nodes/:id` endpoint returns a different JSON envelope
  than `/nodes` (system details under `data`, no top-level `prices`), which the
  SDK previously decoded into the list-shaped struct, leaving every load field
  at zero. Migrated to the SDK's dedicated `NodeDetail` type (go-sdk `v0.4.0`)
  and added a reliability row.

### Changed

- Bumped `github.com/octaspace/go-sdk` to `v0.4.0`, whose `Nodes.Find` now
  returns `*NodeDetail`. `octa nodes show` no longer prints node prices — the
  detail endpoint does not return them; use `octa nodes list` for prices.

## [0.1.0] - 2026-07-22

### Added

- Node management commands covering the full `/nodes` API surface:
  `octa nodes show <id>`, `nodes ident <id>`, `nodes logs <id>`,
  `nodes prices <id>`, and `nodes reboot <id>`. `nodes list` gains `--country`
  and `--app` filters.
- `octa account wallet` generates a new wallet for the authenticated account.
- `octa sessions info <uuid-or-prefix>` shows detailed information for a single
  session (`-o json` prints the raw server response).
- `octa idle-jobs show <node-id> <job-id>` and
  `octa idle-jobs logs <node-id> <job-id>` inspect idle jobs.
- `octa network` prints network statistics and configuration.

### Changed

- CI and release workflows updated to the current major versions of the GitHub
  Actions they use (`checkout`, `setup-go`, `upload-artifact`,
  `download-artifact`, `golangci-lint-action`, `action-gh-release`) to clear the
  deprecated Node 20 runtime warnings.

### Upgrade notes

- No breaking changes. All additions are new commands; existing command
  behavior and JSON output are unchanged.
- The `render` service (`/services/r`, `/services/render`) is intentionally not
  exposed yet and is planned for a later release.

## [0.0.8] - 2026-07-22

### Added

- Integrated WireGuard VPN daemon: run `sudo octa daemon` so that
  `octa vpn connect` brings up and tears down the local operating-system tunnel.
  The CLI records the session UUID only after the tunnel is up and attempts a
  bounded cleanup of the created server session if readiness, tunnel setup, or
  config saving fails.
- Documentation suite under `docs/` (getting started, compute, VPN, command
  reference, JSON output, and development).
- Offline test suite for command handlers, the SDK client wrapper, and table
  rendering; it runs without an API token and makes no live requests.
- Continuous-integration workflow (`.github/workflows/ci.yml`) running build,
  vet, gofmt, tests, race tests, golangci-lint, and a five-target cross-build on
  pull requests and pushes to `main`.
- `.golangci.yml` linter configuration and this changelog.
- Release workflow now publishes a `SHA256SUMS` checksum file alongside the
  platform binaries.

### Changed

- Consolidated the project into a single monorepo binary built from
  `./cmd/octa`.
- Replaced the bespoke `internal/api` HTTP client with the official OctaSpace Go
  SDK (`github.com/octaspace/go-sdk v0.3.0`). Command handlers now call
  resource-scoped SDK methods, and `internal/client` maps common typed API
  errors to CLI messages.
- `-o json` prints the server-shaped API response through the SDK's scoped raw
  methods. Treat it as an API-server contract, not a versioned CLI schema:
  consumers must tolerate new fields.

### Fixed

- Release workflow builds `./cmd/octa` and injects the version into
  `github.com/octaspace/octa/cli.version`. The previous configuration built the
  repository root (which contains no Go files) and set `cmd.version`, so release
  binaries failed to build or reported `dev` instead of the pushed tag.

### Upgrade notes

- WireGuard (`octa vpn connect`, the default protocol) now requires the local
  privileged daemon: start `sudo octa daemon` in a separate terminal and keep it
  running. Shadowsocks, OpenVPN, and V2Ray sessions do not use the local tunnel.
- Build from source with `make build` (Go 1.25 or later); the entry point is
  `./cmd/octa`, not the repository root.

[0.1.0]: https://github.com/octaspace/octa-cli/releases/tag/v0.1.0
[0.0.8]: https://github.com/octaspace/octa-cli/releases/tag/v0.0.8
