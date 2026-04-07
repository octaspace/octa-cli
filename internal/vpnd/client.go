package vpnd

import (
	"encoding/json"
	"fmt"
	"net"
	"runtime"
)

func SocketPath() string {
	if runtime.GOOS == "windows" {
		return `C:\ProgramData\octa-vpn\octa-vpn.sock`
	}
	return "/var/run/octa-vpn.sock"
}

func send(req Request) (*Response, error) {
	conn, err := net.Dial("unix", SocketPath())
	if err != nil {
		return nil, fmt.Errorf("cannot connect to octa-vpnd (is it running?): %w", err)
	}
	defer conn.Close()

	if err := json.NewEncoder(conn).Encode(req); err != nil {
		return nil, fmt.Errorf("send: %w", err)
	}

	var resp Response
	if err := json.NewDecoder(conn).Decode(&resp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return &resp, nil
}

// Connect sends a WireGuard config to the daemon and brings up the tunnel.
func Connect(config string) (*Response, error) {
	return send(Request{Cmd: "connect", Config: config})
}

// Disconnect tears down the active tunnel.
func Disconnect() (*Response, error) {
	return send(Request{Cmd: "disconnect"})
}

// Status returns the current tunnel state.
func Status() (*Response, error) {
	return send(Request{Cmd: "status"})
}
