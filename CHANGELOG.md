# Changelog

All notable changes to this project will be documented in this file.

The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).
This project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

Changelog tracking starts with `0.0.8`; earlier tags (`v0.0.1`–`v0.0.7`) are
recorded only through their Git history and GitHub releases.

---

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

[0.0.8]: https://github.com/octaspace/octa-cli/releases/tag/v0.0.8
