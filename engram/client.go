package engram

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
)

const defaultBaseURL = "http://127.0.0.1:8280"

type Client struct {
	baseURL    string
	httpClient *http.Client
	appID      string
	userID     string
}

type Option func(*Client)

func WithBaseURL(url string) Option { return func(c *Client) { c.baseURL = url } }
func WithAppID(id string) Option    { return func(c *Client) { c.appID = id } }
func WithUserID(id string) Option   { return func(c *Client) { c.userID = id } }

func NewClient(opts ...Option) *Client {
	c := &Client{
		baseURL:    defaultBaseURL,
		httpClient: &http.Client{Timeout: 10 * time.Second},
		appID:      "cursor-global-kb",
		userID:     "nfsarch33",
	}
	if u := os.Getenv("ENGRAM_BASE_URL"); u != "" {
		c.baseURL = u
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

type Memory struct {
	ID      string `json:"id"`
	Content string `json:"content"`
	Score   float64 `json:"score,omitempty"`
}

func (c *Client) Add(content string) (string, error) {
	body := map[string]any{
		"messages": []map[string]string{{"role": "user", "content": content}},
		"user_id":  c.userID,
		"app_id":   c.appID,
		"infer":    false,
	}
	data, _ := json.Marshal(body)
	resp, err := c.httpClient.Post(c.baseURL+"/v1/memories/", "application/json", bytes.NewReader(data))
	if err != nil {
		return "", fmt.Errorf("engram add: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("engram add: status %d", resp.StatusCode)
	}
	var result struct{ ID string `json:"id"` }
	json.NewDecoder(resp.Body).Decode(&result)
	return result.ID, nil
}

func (c *Client) Search(query string, limit int) ([]Memory, error) {
	if limit <= 0 {
		limit = 5
	}
	body := map[string]any{
		"query":   query,
		"user_id": c.userID,
		"app_id":  c.appID,
		"limit":   limit,
	}
	data, _ := json.Marshal(body)
	resp, err := c.httpClient.Post(c.baseURL+"/v1/memories/search/", "application/json", bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("engram search: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("engram search: status %d", resp.StatusCode)
	}
	var result struct{ Results []Memory `json:"results"` }
	json.NewDecoder(resp.Body).Decode(&result)
	return result.Results, nil
}

func (c *Client) Healthz() error {
	resp, err := c.httpClient.Get(c.baseURL + "/healthz")
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("engram unhealthy: %d", resp.StatusCode)
	}
	return nil
}
