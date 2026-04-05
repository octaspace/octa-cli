package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const baseURL = "https://api.octa.space"

// Client is an HTTP client for the OctaSpace API.
type Client struct {
	apiKey     string
	httpClient *http.Client
}

// NewClient creates a new API client with the given API key.
func NewClient(apiKey string) *Client {
	return &Client{
		apiKey: apiKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// get performs a GET request to the given path and decodes the JSON response into dst.
func (c *Client) get(path string, dst interface{}) error {
	req, err := http.NewRequest(http.MethodGet, baseURL+path, nil)
	if err != nil {
		return fmt.Errorf("could not create request: %w", err)
	}

	req.Header.Set("authorization", c.apiKey)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("could not read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	if err := json.Unmarshal(body, dst); err != nil {
		return fmt.Errorf("could not parse response: %w", err)
	}

	return nil
}

// post performs a POST request with a JSON body and decodes the JSON response into dst.
func (c *Client) post(path string, body interface{}, dst interface{}) error {
	data, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("could not encode request body: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, baseURL+path, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("could not create request: %w", err)
	}

	req.Header.Set("authorization", c.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("could not read response body: %w", err)
	}

	if resp.StatusCode >= 400 {
		return fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	if err := json.Unmarshal(respBody, dst); err != nil {
		return fmt.Errorf("could not parse response: %w", err)
	}

	return nil
}

// getRaw performs a GET request and returns the raw response bytes.
func (c *Client) getRaw(path string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, baseURL+path, nil)
	if err != nil {
		return nil, fmt.Errorf("could not create request: %w", err)
	}

	req.Header.Set("authorization", c.apiKey)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("could not read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	return body, nil
}
