# Getting started

## Authenticate

Create an API token in your OctaSpace account, then pass it to `octa auth`:

```bash
octa auth <token>
```

The command calls the account endpoint before saving the token. Authentication,
not-found, and rate-limit responses are presented as concise CLI errors.
Network and server failures retain their original diagnostic message.

The config path is `$XDG_CONFIG_HOME/octa/config.yaml`. If
`XDG_CONFIG_HOME` is unset, the path is `~/.config/octa/config.yaml`. On Unix,
the containing directory is created with mode `0700`; the config file uses mode
`0600`.

Do not pass a token through shell history, shared logs, or command examples.
Prefer a short-lived shell variable when practical:

```bash
read -rs OCTA_TOKEN
printf '\n'
octa auth "$OCTA_TOKEN"
unset OCTA_TOKEN
```

## First session

List available machines and application templates:

```bash
octa compute
octa compute apps
```

Deploy an app to a selected node:

```bash
octa compute deploy --app <app-uuid> --node <node-id>
```

The command prints every accepted session UUID. Confirm the result and stop it
when finished:

```bash
octa sessions
octa sessions stop <session-uuid-or-unique-prefix>
```

## Cancellation and errors

The root command passes `SIGINT` and `SIGTERM` through a context, so `Ctrl-C`
cancels in-flight API calls and VPN readiness polling. A successful create is
not retried automatically; this prevents accidental duplicate paid sessions.

For request details or API fields not represented in a table, use `-o json`.
See [JSON output](json-output.md).
