package api

// VPNRelay represents an available VPN relay node from GET /services/vpn.
type VPNRelay struct {
	NodeID        int     `json:"node_id"`
	Country       string  `json:"country"`
	City          string  `json:"city"`
	PricePerGB    float64 `json:"traffic_price_usd"`
	DownloadSpeed int64   `json:"net_down_mbits"`
	UploadSpeed   int64   `json:"net_up_mbits"`
	Residential   bool    `json:"residential"`
}

// VPNConnectRequest is the body for POST /services/vpn.
type VPNConnectRequest struct {
	NodeID  int    `json:"node_id"`
	SubKind string `json:"subkind"`
}

// VPNConnectResponse is the response from POST /services/vpn.
type VPNConnectResponse struct {
	UUID string `json:"uuid"`
}

// ConnectVPN creates a VPN session on the given node.
func (c *Client) ConnectVPN(nodeID int, protocol string) (*VPNConnectResponse, error) {
	var resp VPNConnectResponse
	if err := c.post("/services/vpn", VPNConnectRequest{
		NodeID:  nodeID,
		SubKind: protocol,
	}, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// ListVPNRelays fetches available VPN relay nodes from GET /services/vpn.
func (c *Client) ListVPNRelays() ([]VPNRelay, error) {
	var relays []VPNRelay
	if err := c.get("/services/vpn", &relays); err != nil {
		return nil, err
	}
	return relays, nil
}

// ListVPNRelaysRaw fetches VPN relays and returns raw JSON bytes.
func (c *Client) ListVPNRelaysRaw() ([]byte, error) {
	return c.getRaw("/services/vpn")
}
