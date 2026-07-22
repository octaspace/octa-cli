# VPN

## Choose a relay

```bash
octa vpn relay list
octa vpn relay search kyiv
octa vpn relay search ukraine --residential
octa vpn relay set <node-id>
octa vpn relay get
```

`relay set` verifies that the node exists and stores its ID, city, and country
in the local config. `vpn connect` and `vpn status` use that relay selection.

## WireGuard tunnel

WireGuard is the default protocol. It needs a local privileged daemon to create
and remove the operating-system tunnel. Run the daemon in a separate terminal
and keep it running while using WireGuard:

```bash
sudo octa daemon
```

Then create the VPN session and tunnel:

```bash
octa vpn connect
# equivalent: octa vpn connect --protocol wg
```

The CLI waits for the API configuration, sends it to the daemon, and records the
created session UUID only after the tunnel is up. If readiness, tunnel setup, or
config saving fails, it attempts to stop the newly created API session. `Ctrl-C`
cancels the wait promptly; the cleanup uses its own bounded context.

Disconnect both the local tunnel and the recorded server session:

```bash
octa vpn disconnect
```

If stopping the server session fails, its UUID stays in the config so a later
`vpn disconnect` can retry. A session already absent on the server is treated as
stopped.

## Other VPN types

The API also supports Shadowsocks, OpenVPN, and V2Ray. These commands create a
server session and print its UUID; they do not configure a local tunnel.

```bash
octa vpn connect --protocol ss
octa vpn connect --protocol openvpn
octa vpn connect --protocol v2ray --v2ray-protocol vless
```

V2Ray requires one of `vless`, `vmess`, or `trojan` in `--v2ray-protocol`.
Inspect the active VPN configuration at the selected relay with
`vpn status --config`, or display a QR code with `vpn status --qr`.

## Status and machine output

```bash
octa vpn status
octa vpn status --config
octa vpn status --qr
octa vpn status -o json
```

When a saved WireGuard UUID exists, status uses that exact VPN session. Without
one, it searches active VPN sessions at the selected relay and excludes
machine-rental sessions on the same node.
