# JSON output and API compatibility

Most commands with `-o json` pass the SDK's raw resource response through to
stdout, unmodified apart from the CLI's trailing newline:

```bash
octa account -o json
octa nodes list -o json
octa compute -o json
octa compute apps -o json
octa sessions -o json
octa sessions info <uuid> -o json
octa vpn relay list -o json
octa vpn status -o json
```

These keep every API field, including ones that don't exist yet in the SDK's
typed models. They do not decode and re-marshal the payload.

A smaller set of commands has no raw SDK method to pass through — either the
command filters results client-side (`compute search`, `vpn relay search`) or
the endpoint has no `*Raw` accessor in the SDK (`nodes show`, `compute logs`).
For these, `-o json` marshals the SDK's typed model instead:

```bash
octa nodes show <id> -o json
octa compute search <query> -o json
octa compute logs <uuid> -o json
octa vpn relay search <query> -o json
```

Typed re-marshaling can drop fields the API added since the model was last
updated — that's a real risk, not a theoretical one: `nodes show` and
`sessions`/`vpn status` both hit exactly this class of bug historically (see
`CHANGELOG.md` v0.1.1 and v0.2.0, and the go-sdk contract-fixture tests these
models are now checked against). Prefer the endpoints above with a raw form
when a script needs a guaranteed-complete server response.

Mutating and local-state commands (`nodes prices`, `nodes reboot`,
`compute deploy`, `sessions stop`, `vpn relay set`, `vpn relay get`,
`vpn connect`, `vpn disconnect`, `auth`) don't have a single server resource
to dump; `-o json` on these prints a small derived confirmation object
(status, affected ID, and similar) rather than an API passthrough.

The JSON shape of raw-passthrough commands is controlled by the OctaSpace API
and may evolve independently of CLI releases. Scripts should tolerate unknown
fields, validate the fields they consume, and pin an API-side contract where a
strict schema is required.

Table output is different by design: it uses typed SDK models, normalizes
known API inconsistencies, and selects a concise set of human-readable fields.

For API endpoint definitions, consult the [OctaSpace API documentation](https://api.octa.space/api-docs). For SDK model and transport behaviour, see the
[Go SDK documentation](https://github.com/octaspace/go-sdk/tree/main/docs).
