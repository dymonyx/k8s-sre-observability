package httpx

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Client struct {
	baseURL string
	client  *http.Client
	logger  *slog.Logger
}

func NewClient(baseURL string, timeout time.Duration, logger *slog.Logger) *Client {
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		client:  &http.Client{Timeout: timeout},
		logger:  logger,
	}
}

func (c *Client) GetJSON(ctx context.Context, path string, query url.Values, target any) error {
	endpoint, err := url.Parse(c.baseURL + path)
	if err != nil {
		return fmt.Errorf("parse endpoint: %w", err)
	}
	endpoint.RawQuery = query.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	if requestID := RequestIDFromContext(ctx); requestID != "" {
		req.Header.Set("X-Request-ID", requestID)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("GET %s: %w", endpoint.Redacted(), err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return fmt.Errorf("read response body: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("GET %s returned status %d: %s", endpoint.Redacted(), resp.StatusCode, strings.TrimSpace(string(body)))
	}
	if err := json.Unmarshal(body, target); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	return nil
}
