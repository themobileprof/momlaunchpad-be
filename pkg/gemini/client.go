package gemini

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

// HTTPClient implements the llm.Client interface for Gemini using REST API
type HTTPClient struct {
	apiKey     string
	baseURL    string
	model      string
	httpClient *http.Client
	timeout    time.Duration
}

// Ensure HTTPClient implements llm.Client
var _ llm.Client = (*HTTPClient)(nil)

// Config holds configuration for the Gemini client
type Config struct {
	APIKey  string
	Model   string        // Default: gemini-pro
	Timeout time.Duration // Default: 30s
}

// NewHTTPClient creates a new Gemini HTTP client
func NewHTTPClient(config Config) *HTTPClient {
	if config.Model == "" {
		config.Model = "gemini-2.0-flash"
	}
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}

	// Optimized transport
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
		baseURL: "https://generativelanguage.googleapis.com/v1beta/models",
		model:   config.Model,
		httpClient: &http.Client{
			Timeout:   config.Timeout,
			Transport: transport,
		},
		timeout: config.Timeout,
	}
}

// Internal Gemini types
type geminiRequest struct {
	Contents         []geminiContent  `json:"contents"`
	GenerationConfig generationConfig `json:"generationConfig,omitempty"`
}

type geminiContent struct {
	Role  string       `json:"role"`
	Parts []geminiPart `json:"parts"`
}

type geminiPart struct {
	Text string `json:"text"`
}

type generationConfig struct {
	Temperature     float64 `json:"temperature,omitempty"`
	MaxOutputTokens int     `json:"maxOutputTokens,omitempty"`
}

type geminiResponse struct {
	Candidates []struct {
		Content struct {
			Parts []geminiPart `json:"parts"`
		} `json:"content"`
		FinishReason string `json:"finishReason"`
	} `json:"candidates"`
	UsageMetadata struct {
		PromptTokenCount     int `json:"promptTokenCount"`
		CandidatesTokenCount int `json:"candidatesTokenCount"`
		TotalTokenCount      int `json:"totalTokenCount"`
	} `json:"usageMetadata"`
}

// Helper to convert llm.ChatRequest to Gemini request
func (c *HTTPClient) toGeminiRequest(req llm.ChatRequest) geminiRequest {
	contents := make([]geminiContent, len(req.Messages))
	for i, msg := range req.Messages {
		role := msg.Role
		if role == "assistant" {
			role = "model"
		} else if role == "system" {
			// Gemini doesn't always support system role in contents depending on version, 
			// but modern models do or we can prepend to first user message.
			// Ideally we use specialized systemInstruction field, but for simplicity/general compatibility 
			// let's try mapping system to user with a prefix or just user if model supports it.
			// Actually gemini-1.5-pro supports system instructions, but older gemini-pro might not.
			// Let's coerce system to "user" for maximum compatibility in this simple REST implementation
			// or assume user is using a model that handles it. 
			// Safest fallback for "gemini-pro" is usually prepending to prompt or using "user".
			role = "user" 
		}

		contents[i] = geminiContent{
			Role: role,
			Parts: []geminiPart{
				{Text: msg.Content},
			},
		}
	}

	return geminiRequest{
		Contents: contents,
		GenerationConfig: generationConfig{
			Temperature:     req.Temperature,
			MaxOutputTokens: req.MaxTokens,
		},
	}
}

// StreamChatCompletion implements llm.Client.StreamChatCompletion
func (c *HTTPClient) StreamChatCompletion(ctx context.Context, req llm.ChatRequest) (<-chan llm.ChatChunk, error) {
	// Gemini streaming endpoint: ...:streamGenerateContent
	url := fmt.Sprintf("%s/%s:streamGenerateContent?key=%s&alt=sse", c.baseURL, c.model, c.apiKey)
	
	gemReq := c.toGeminiRequest(req)
	body, err := json.Marshal(gemReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	ch := make(chan llm.ChatChunk, 32)

	go func() {
		defer close(ch)
		defer resp.Body.Close()

		reader := bufio.NewReader(resp.Body)
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				if err != io.EOF {
					// Log error? 
				}
				break
			}

			line = strings.TrimSpace(line)
			if !strings.HasPrefix(line, "data: ") {
				continue
			}

			data := strings.TrimPrefix(line, "data: ")
			if data == "[DONE]" {
				break
			}

			var gResp geminiResponse
			if err := json.Unmarshal([]byte(data), &gResp); err != nil {
				continue
			}

			// Convert to llm.ChatChunk
			if len(gResp.Candidates) > 0 {
				content := ""
				if len(gResp.Candidates[0].Content.Parts) > 0 {
					content = gResp.Candidates[0].Content.Parts[0].Text
				}
				
				chunk := llm.ChatChunk{
					Model: c.model,
					Choices: []struct {
						Index        int     `json:"index"`
						Delta        llm.Delta   `json:"delta"`
						FinishReason *string `json:"finish_reason"`
					}{
						{
							Delta: llm.Delta{
								Content: content,
							},
						},
					},
				}
				
				select {
				case ch <- chunk:
				case <-ctx.Done():
					return
				}
			}
		}
	}()

	return ch, nil
}

// ChatCompletion implements llm.Client.ChatCompletion
func (c *HTTPClient) ChatCompletion(ctx context.Context, req llm.ChatRequest) (*llm.ChatResponse, error) {
	// Gemini content generation endpoint: ...:generateContent
	url := fmt.Sprintf("%s/%s:generateContent?key=%s", c.baseURL, c.model, c.apiKey)

	gemReq := c.toGeminiRequest(req)
	body, err := json.Marshal(gemReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	var gResp geminiResponse
	if err := json.NewDecoder(resp.Body).Decode(&gResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Map to llm.ChatResponse
	content := ""
	finishReason := ""
	if len(gResp.Candidates) > 0 {
		if len(gResp.Candidates[0].Content.Parts) > 0 {
			content = gResp.Candidates[0].Content.Parts[0].Text
		}
		finishReason = gResp.Candidates[0].FinishReason
	}

	return &llm.ChatResponse{
		Model: c.model,
		Choices: []struct {
			Index   int `json:"index"`
			Message struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			} `json:"message"`
			FinishReason string `json:"finish_reason"`
		}{
			{
				Message: struct {
					Role    string `json:"role"`
					Content string `json:"content"`
				}{
					Role:    "assistant",
					Content: content,
				},
				FinishReason: finishReason,
			},
		},
		Usage: struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
			TotalTokens      int `json:"total_tokens"`
		}{
			PromptTokens:     gResp.UsageMetadata.PromptTokenCount,
			CompletionTokens: gResp.UsageMetadata.CandidatesTokenCount,
			TotalTokens:      gResp.UsageMetadata.TotalTokenCount,
		},
	}, nil
}
