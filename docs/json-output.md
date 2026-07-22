# JSON output and API compatibility

Commands with `-o json` pass the SDK's raw resource response through to stdout:

```bash
octa account -o json
octa nodes list -o json
octa compute -o json
octa compute apps -o json
octa sessions -o json
octa vpn relay list -o json
octa vpn status -o json
```

The response keeps API fields that may not exist yet in the SDK's typed models,
including future additions such as application template commands or session
detail fields. The CLI appends its normal trailing newline; otherwise it does
not decode and re-marshal the payload.

The JSON shape is controlled by the OctaSpace API and may evolve independently
of CLI releases. Scripts should tolerate unknown fields, validate the fields
they consume, and pin an API-side contract where a strict schema is required.

Table output is different by design: it uses typed SDK models, normalizes known
API inconsistencies, and selects a concise set of human-readable fields.

For API endpoint definitions, consult the [OctaSpace API documentation](https://api.octa.space/api-docs). For SDK model and transport behaviour, see the
[Go SDK documentation](https://github.com/octaspace/go-sdk/tree/main/docs).
