// Package client provides a thin HTTP client for the krypt API.
// It never transmits the encryption key – that stays in the URL fragment.
package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Client talks to a krypt backend.
type Client struct {
	base    string       // e.g. "https://paste.example.com"
	http    *http.Client
}

// New returns a Client for the given server base URL (no trailing slash).
func New(base string) *Client {
	return &Client{
		base: base,
		http: &http.Client{Timeout: 30 * time.Second},
	}
}

// CreateRequest is the POST /api/paste body.
type CreateRequest struct {
	EncryptedData string `json:"encrypted_data"`
	IV            string `json:"iv"`
	Salt          string `json:"salt"`
	ExpiresIn     *int   `json:"expires_in"`
	BurnAfterRead bool   `json:"burn_after_read"`
	HasPassword   bool   `json:"has_password"`
}

// CreateResponse is the POST /api/paste response.
type CreateResponse struct {
	ID          string `json:"id"`
	DeleteToken string `json:"delete_token"`
}

// PasteData is the GET /api/paste/{id} response.
type PasteData struct {
	EncryptedData string `json:"encrypted_data"`
	IV            string `json:"iv"`
	Salt          string `json:"salt"`
	BurnAfterRead bool   `json:"burn_after_read"`
	HasPassword   bool   `json:"has_password"`
	CreatedAt     int64  `json:"created_at"`
	ExpiresAt     *int64 `json:"expires_at"`
	ReadCount     int    `json:"read_count"`
}

// CreatePaste posts encrypted data to the server and returns the paste ID and delete token.
func (c *Client) CreatePaste(req CreateRequest) (CreateResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return CreateResponse{}, err
	}

	resp, err := c.http.Post(c.base+"/api/paste", "application/json", bytes.NewReader(body))
	if err != nil {
		return CreateResponse{}, fmt.Errorf("POST /api/paste: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		var apiErr struct{ Error string `json:"error"` }
		_ = json.NewDecoder(resp.Body).Decode(&apiErr)
		return CreateResponse{}, fmt.Errorf("server returned %d: %s", resp.StatusCode, apiErr.Error)
	}

	var cr CreateResponse
	if err := json.NewDecoder(resp.Body).Decode(&cr); err != nil {
		return CreateResponse{}, fmt.Errorf("decode response: %w", err)
	}
	return cr, nil
}

// GetPaste retrieves encrypted paste data by ID.
func (c *Client) GetPaste(id string) (*PasteData, error) {
	req, err := http.NewRequest(http.MethodGet, c.base+"/api/paste/"+id, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Cache-Control", "no-cache")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("GET /api/paste/%s: %w", id, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var apiErr struct{ Error string `json:"error"` }
		_ = json.NewDecoder(resp.Body).Decode(&apiErr)
		return nil, fmt.Errorf("server returned %d: %s", resp.StatusCode, apiErr.Error)
	}

	var pd PasteData
	if err := json.NewDecoder(resp.Body).Decode(&pd); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return &pd, nil
}
