# Getting started

## Authenticate

Create an API key in OctaSpace under **Profile → Credentials**. It starts with
`octa_ak_` and is shown only once.

```bash
octa auth <token>
```

The command reads the account balance before saving the token. It is sent as
`Authorization: Bearer <token>`.

The config path is `$XDG_CONFIG_HOME/octa/next.yaml`. If
`XDG_CONFIG_HOME` is unset, the path is `~/.config/octa/next.yaml`. On Unix,
the containing directory is created with mode `0700`; the config file uses mode
`0600`.

A key that does not start with `octa_ak_` is rejected before any request.

Do not pass a token through shell history, shared logs, or command examples.
Prefer a short-lived shell variable when practical:

```bash
read -rs OCTA_TOKEN
printf '\n'
octa auth "$OCTA_TOKEN"
unset OCTA_TOKEN
```

## API host

The default host is `https://api-dev.octa.space`. To use another host, set `OCTA_API_URL` or `api_url` in the config file
(the environment variable wins):

```yaml
api_key: octa_ak_...
api_url: https://api-dev.octa.space
```

## First session

```bash
octa compute
octa compute start --node <number> --image ubuntu:24.04 --ssh --wait
octa compute connect <session-uuid-or-unique-prefix>
```

Stop it when finished:

```bash
octa sessions
octa sessions stop <session-uuid-or-unique-prefix>
```

## Cancellation and errors

The root command passes `SIGINT` and `SIGTERM` through a context, so `Ctrl-C`
cancels in-flight API calls and `--wait` polling. Starts and restarts are not
retried automatically; this prevents accidental duplicate paid sessions.

API errors are shown with the API's own message (for example
`Insufficient balance` or `Session is still starting or restarting`); rate
limits show how long to wait.

For API fields not represented in a table, use `-o json`.
See [JSON output](json-output.md).
