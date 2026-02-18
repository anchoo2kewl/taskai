package yjs

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"go.uber.org/zap"
)

// Client is an HTTP client for the Yjs processor microservice
type Client struct {
	baseURL    string
	httpClient *http.Client
	logger     *zap.Logger
}

// NewClient creates a new Yjs processor client
func NewClient(baseURL string, logger *zap.Logger) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger: logger,
	}
}

// ApplyUpdatesRequest represents the request to apply Yjs updates
type ApplyUpdatesRequest struct {
	Updates []string `json:"updates"`
}

// ApplyUpdatesResponse represents the response from applying updates
type ApplyUpdatesResponse struct {
	State string `json:"state"`
}

// ApplyUpdates applies an array of Yjs updates and returns the current state
func (c *Client) ApplyUpdates(ctx context.Context, updates []string) (string, error) {
	reqBody := ApplyUpdatesRequest{
		Updates: updates,
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/apply-updates", bytes.NewReader(bodyBytes))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	c.logger.Debug("Calling yjs-processor /apply-updates",
		zap.Int("update_count", len(updates)),
	)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("yjs-processor returned status %d: %s", resp.StatusCode, string(respBody))
	}

	var response ApplyUpdatesResponse
	if err := json.Unmarshal(respBody, &response); err != nil {
		return "", fmt.Errorf("failed to unmarshal response: %w", err)
	}

	c.logger.Debug("Successfully applied Yjs updates",
		zap.Int("state_size", len(response.State)),
	)

	return response.State, nil
}

// ExtractBlocksRequest represents the request to extract blocks from a Yjs document
type ExtractBlocksRequest struct {
	State string `json:"state"`
}

// Block represents a content block extracted from a Yjs document
type Block struct {
	Type          string `json:"type"`
	Level         *int   `json:"level"`
	HeadingsPath  string `json:"headings_path"`
	PlainText     string `json:"plain_text"`
	CanonicalJSON string `json:"canonical_json"`
	Position      int    `json:"position"`
}

// ExtractBlocksResponse represents the response from extracting blocks
type ExtractBlocksResponse struct {
	Blocks []Block `json:"blocks"`
}

// ExtractBlocks extracts content blocks from a Yjs document state for indexing
func (c *Client) ExtractBlocks(ctx context.Context, state string) ([]Block, error) {
	reqBody := ExtractBlocksRequest{
		State: state,
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/extract-blocks", bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	c.logger.Debug("Calling yjs-processor /extract-blocks",
		zap.Int("state_size", len(state)),
	)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("yjs-processor returned status %d: %s", resp.StatusCode, string(respBody))
	}

	var response ExtractBlocksResponse
	if err := json.Unmarshal(respBody, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	c.logger.Debug("Successfully extracted blocks",
		zap.Int("block_count", len(response.Blocks)),
	)

	return response.Blocks, nil
}

// HealthCheck checks if the yjs-processor service is healthy
func (c *Client) HealthCheck(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/health", nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("health check failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}
