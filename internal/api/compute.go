package api

// MachineRental represents an available machine for rent from GET /services/mr.
type MachineRental struct {
	NodeID        int    `json:"node_id"`
	Country       string `json:"country"`
	City          string `json:"city"`
	CPUModelName  string `json:"cpu_model_name"`
	CPUCores      int    `json:"cpu_cores"`
	CPUVendorID   string `json:"cpu_vendor_id"`
	TotalMemory   int64  `json:"total_memory"`
	FreeDisk      int64  `json:"free_disk"`
	CUDAVersion   string `json:"cuda_version"`
	TotalPriceUSD float64 `json:"total_price_usd"`
	BaseUSD       int     `json:"base_usd"`
	StorageUSD    int     `json:"storage_usd"`
	TrafficUSD    int     `json:"traffic_usd"`

	GPUs []struct {
		Model      string `json:"model"`
		MemTotalMB int    `json:"mem_total_mb"`
	} `json:"gpus"`
}

// DeployRequest is the body for POST /services/mr.
type DeployRequest struct {
	App    string `json:"app"`
	NodeID int64  `json:"node_id"`
	Image  string `json:"image,omitempty"`
}

// DeployResponse is the response from POST /services/mr.
type DeployResponse struct {
	UUID string `json:"uuid"`
}

// DeployMachine starts a machine rental service.
func (c *Client) DeployMachine(req DeployRequest) (*DeployResponse, error) {
	var resp DeployResponse
	if err := c.post("/services/mr", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// ListMachinesForRent fetches available machines for rent from GET /services/mr.
func (c *Client) ListMachinesForRent() ([]MachineRental, error) {
	var machines []MachineRental
	if err := c.get("/services/mr", &machines); err != nil {
		return nil, err
	}
	return machines, nil
}

// ListMachinesForRentRaw fetches available machines and returns raw JSON bytes.
func (c *Client) ListMachinesForRentRaw() ([]byte, error) {
	return c.getRaw("/services/mr")
}
