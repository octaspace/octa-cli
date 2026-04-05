package api

// App represents an available application.
type App struct {
	UUID     string `json:"uuid"`
	Name     string `json:"name"`
	Image    string `json:"image"`
	Category string `json:"category"`
}

// ListApps fetches available applications from GET /apps.
func (c *Client) ListApps() ([]App, error) {
	var apps []App
	if err := c.get("/apps", &apps); err != nil {
		return nil, err
	}
	return apps, nil
}

// ListAppsRaw fetches available applications and returns raw JSON bytes.
func (c *Client) ListAppsRaw() ([]byte, error) {
	return c.getRaw("/apps")
}
