# Changelog

All notable changes to this project will be documented in this file.

The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).
This project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

Changelog tracking starts with `0.0.8`; earlier tags (`v0.0.1`–`v0.0.7`) are
recorded only through their Git history and GitHub releases.

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
