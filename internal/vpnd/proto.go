package vpnd

// Request is sent by the client to the daemon over the Unix socket.
type Request struct {
	Cmd    string `json:"cmd"`              // "connect" | "disconnect" | "status"
	Config string `json:"config,omitempty"` // WireGuard config text (connect only)
}

// Response is returned by the daemon.
type Response struct {
	OK        bool   `json:"ok"`
	Interface string `json:"interface,omitempty"` // active interface name, if any
	Error     string `json:"error,omitempty"`
}
