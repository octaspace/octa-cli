package vpnd

// ActiveTunnel is the interface for an active VPN tunnel.
type ActiveTunnel interface {
	Interface() string
	Close()
}
