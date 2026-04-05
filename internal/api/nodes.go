package api

// Node represents a compute node in the OctaSpace network.
type Node struct {
	ID     int64  `json:"id"`
	IP     string `json:"ip"`
	State  string `json:"state"`
	OSN    string `json:"osn"`
	Uptime int64  `json:"uptime"`
	VrfDt  string `json:"vrf_dt"`

	Location struct {
		City    string `json:"city"`
		Country string `json:"country"`
	} `json:"location"`

	Prices struct {
		Base    int `json:"base"`
		Storage int `json:"storage"`
		Traffic int `json:"traffic"`
	} `json:"prices"`

	System struct {
		Arch           string  `json:"arch"`
		CPUCores       int     `json:"cpu_cores"`
		CPULoadPercent float64 `json:"cpu_load_percent"`
		CPUModelName   string  `json:"cpu_model_name"`
		OSVersion      string  `json:"os_version"`

		Disk struct {
			Free        int64   `json:"free"`
			Size        int64   `json:"size"`
			UsedPercent float64 `json:"used_percent"`
		} `json:"disk"`

		GPUs []struct {
			Model      string `json:"model"`
			MemTotalMB int    `json:"mem_total_mb"`
			MemFreeMB  int    `json:"mem_free_mb"`
		} `json:"gpus"`

		Memory struct {
			Free int64 `json:"free"`
			Size int64 `json:"size"`
		} `json:"memory"`
	} `json:"system"`
}

// ListNodes fetches all nodes from the API.
func (c *Client) ListNodes() ([]Node, error) {
	var nodes []Node
	if err := c.get("/nodes", &nodes); err != nil {
		return nil, err
	}
	return nodes, nil
}

// ListNodesRaw fetches all nodes and returns the raw JSON bytes.
func (c *Client) ListNodesRaw() ([]byte, error) {
	return c.getRaw("/nodes")
}
