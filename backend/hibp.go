package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"
)

type HIBPClient struct {
	BaseURL string
	HTTP    *http.Client
}

func NewHIBPClient(base string) *HIBPClient {
	return &HIBPClient{BaseURL: base, HTTP: &http.Client{Timeout: 8 * time.Second}}
}

// Range calls GET /range/{prefix}. Add-Padding makes every response a similar
// size so observers can't infer the prefix from response length; padded rows
// have a count of 0 and are ignored by the frontend.
func (c *HIBPClient) Range(ctx context.Context, prefix string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+"/range/"+prefix, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "pwcheck-leak-checker")
	req.Header.Set("Add-Padding", "true")
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("hibp status %d", resp.StatusCode)
	}
	b, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return "", err
	}
	return string(b), nil
}
