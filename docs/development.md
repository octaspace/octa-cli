# Development and verification

The CLI is a thin command layer over `github.com/octaspace/go-sdk v0.3.0`.
HTTP transport, authentication headers, typed errors, request cancellation,
safe GET retry, and API response decoding belong in the SDK rather than command
handlers.

## Build and test

Go 1.25 or later is required by `go.mod`.

```bash
make build
go test ./...
go test -race ./...
go vet ./...
gofmt -l .
git diff --check
```

The test suite uses fakes and fixtures only; it does not require a token or
make live API requests. Avoid adding live, session-creating checks to the
default suite. SDK live contract checks are separately opt-in; see the
[SDK testing guide](https://github.com/octaspace/go-sdk/blob/main/docs/testing.md).

## Design boundaries

- Command handlers load config, parse flags, call resource-scoped SDK methods,
  and render output.
- `internal/client` creates the SDK client and maps common typed API errors to
  CLI messages.
- Human tables use typed SDK models. `-o json` uses scoped raw SDK methods so
  machine-readable output preserves server fields.
- Machine-rental deployment remains on the production-compatible SDK `Create`
  array form. Do not switch it implicitly to `CreateSingle`.

When an API field changes, first update or verify the SDK contract and its
fixtures, then add the smallest CLI mapping or raw-output documentation needed.
