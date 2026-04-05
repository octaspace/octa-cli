package api

// Session represents an active or recent session.
type Session struct {
	UUID      string `json:"uuid"`
	Service   string `json:"service"`
	AppName   string `json:"app_name"`
	AppLogo   string `json:"app_logo"`
	NodeID    int64  `json:"node_id"`
	Progress  string `json:"progress"`
	IsReady   bool   `json:"is_ready"`
	Duration     int64  `json:"duration"`
	StartedAt    int64  `json:"started_at"`
	PublicIP     string `json:"public_ip"`
	ChargeAmount uint64 `json:"charge_amount"`

	SSHDirect struct {
		Host string `json:"host"`
		Port int    `json:"port"`
	} `json:"ssh_direct"`

	SSHProxy struct {
		Host string `json:"host"`
		Port int    `json:"port"`
	} `json:"ssh_proxy"`

	URLs map[string]string `json:"urls"`

	Prices struct {
		BaseUSD    int     `json:"base_usd"`
		CurrencyUSD bool   `json:"currency_usd"`
		MarketPrice float64 `json:"market_price"`
	} `json:"prices"`

	NodeHW struct {
		CPU      string `json:"cpu"`
		CPUCores int    `json:"cpu_cores"`
		GPU      []struct {
			Model      string `json:"model"`
			MemTotalMB int    `json:"mem_total_mb"`
		} `json:"gpu"`
		TotalMemory int64 `json:"total_memory"`
	} `json:"node_hw"`
}

// ListSessions fetches active/recent sessions from GET /sessions.
func (c *Client) ListSessions() ([]Session, error) {
	var sessions []Session
	if err := c.get("/sessions", &sessions); err != nil {
		return nil, err
	}
	return sessions, nil
}

// ListSessionsRaw fetches sessions and returns raw JSON bytes.
func (c *Client) ListSessionsRaw() ([]byte, error) {
	return c.getRaw("/sessions")
}

// SessionInfo holds detailed session information from GET /sessions/:uuid/info.
type SessionInfo struct {
	VPNConfig    string `json:"config"`
	TX           int64  `json:"tx"`
	RX           int64  `json:"rx"`
	ChargeAmount BigInt `json:"charge_amount"`
}

// GetSessionInfo fetches detailed info for a session.
func (c *Client) GetSessionInfo(uuid string) (*SessionInfo, error) {
	var info SessionInfo
	if err := c.get("/services/"+uuid+"/info", &info); err != nil {
		return nil, err
	}
	return &info, nil
}

// GetSessionInfoRaw fetches detailed info for a session and returns raw JSON bytes.
func (c *Client) GetSessionInfoRaw(uuid string) ([]byte, error) {
	return c.getRaw("/services/" + uuid + "/info")
}

// StopSession terminates a session by its full UUID.
func (c *Client) StopSession(uuid string) error {
	_, err := c.getRaw("/services/" + uuid + "/stop")
	return err
}
