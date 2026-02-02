package deepseek

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/themobileprof/momlaunchpad-be/pkg/llm"
)

// HTTPClient implements the llm.Client interface using HTTP requests
type HTTPClient struct {
	apiKey     string
	baseURL    string
	model      string
	httpClient *http.Client
	timeout    time.Duration
}

// Ensure HTTPClient implements llm.Client
var _ llm.Client = (*HTTPClient)(nil)

// Config holds configuration for the DeepSeek client
type Config struct {
	APIKey  string
	BaseURL string        // Default: https://api.deepseek.com/v1
	Model   string        // Default: deepseek-chat
	Timeout time.Duration // Default: 30s
}

// NewHTTPClient creates a new DeepSeek HTTP client
func NewHTTPClient(config Config) *HTTPClient {
	if config.BaseURL == "" {
		config.BaseURL = "https://api.deepseek.com/v1"
	}
	if config.Model == "" {
		config.Model = "deepseek-chat"
	}
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}

	// Optimized transport for high throughput and connection reuse
	transport := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   10 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   10,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		ForceAttemptHTTP2:     true,
	}

	return &HTTPClient{
		apiKey:  config.APIKey,
		baseURL: config.BaseURL,
		model:   config.Model,
		httpClient: &http.Client{
			Timeout:   config.Timeout,
			Transport: transport,
		},
		timeout: config.Timeout,
	}
}

// StreamChatCompletion implements llm.Client.StreamChatCompletion
func (c *HTTPClient) StreamChatCompletion(ctx context.Context, req llm.ChatRequest) (<-chan llm.ChatChunk, error) {
	// Set default model if not provided
	if req.Model == "" {
		req.Model = c.model
	}

	// Force streaming
	req.Stream = true

	// Prepare request body
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)

	// Execute request
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}

	// Check status code
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Create channel for streaming chunks (larger buffer for throughput)
	ch := make(chan llm.ChatChunk, 32)

	// Start goroutine to read streaming response
	go func() {
		defer close(ch)
		defer resp.Body.Close()

		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			line := scanner.Text()

			// Skip empty lines
			if line == "" {
				continue
			}

			// SSE format: "data: {...}"
			if !strings.HasPrefix(line, "data: ") {
				continue
			}

			// Extract JSON data
			data := strings.TrimPrefix(line, "data: ")

			// Check for [DONE] marker
			if data == "[DONE]" {
				break
			}

			// Parse chunk
			var chunk llm.ChatChunk
			if err := json.Unmarshal([]byte(data), &chunk); err != nil {
				// Log error but continue processing
				continue
			}

			// Send chunk to channel
			select {
			case ch <- chunk:
			case <-ctx.Done():
				return
			}
		}
	}()

	return ch, nil
}

// ChatCompletion implements llm.Client.ChatCompletion
func (c *HTTPClient) ChatCompletion(ctx context.Context, req llm.ChatRequest) (*llm.ChatResponse, error) {
	// Set default model if not provided
	if req.Model == "" {
		req.Model = c.model
	}

	// Force non-streaming
	req.Stream = false

	// Prepare request body
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)

	// Execute request
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var chatResp llm.ChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &chatResp, nil
}
