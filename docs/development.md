# Development and verification

The CLI is a thin command layer over `github.com/octaspace/go-sdk`, pinned to
the SDK's `next` branch (the next-generation API) until it is tagged.
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

The test suite uses fakes and fixtures only (`internal/ui/testdata` holds live
responses of the next API, shared with the SDK); it does not require a token or
make live API requests. Avoid adding live, session-creating checks to the
default suite. SDK live contract checks are separately opt-in; see the
[SDK testing guide](https://github.com/octaspace/go-sdk/blob/main/docs/testing.md).

## Design boundaries

- Command handlers load config, parse flags, call resource-scoped SDK methods,
  and render output.
- `internal/client` creates the SDK client and maps common typed API errors to
  CLI messages.
- Human tables use typed SDK models. `-o json` prefers scoped raw SDK methods
  so machine-readable output preserves server fields; where no raw method
  exists (client-side search, endpoints without a `*Raw` accessor) it falls
  back to marshaling the typed model — see `docs/json-output.md` for which
  commands fall into each category.
- `compute start` and `render start` use the single-object SDK `Start` for one
  node, which returns the API's rejection reason as an error, and
  `StartBatch` for several nodes or `--from-file`, reporting each result.
- The CLI covers the authenticated API. The public statistics endpoints
  (`/network`, `/stats/gpu/*`) stay in the SDK and are not exposed as commands.

When an API field changes, first update or verify the SDK contract and its
fixtures, then add the smallest CLI mapping or raw-output documentation needed.

To update the SDK to the latest commit of its `next` branch:

```bash
go get github.com/octaspace/go-sdk@next
go mod tidy
```
