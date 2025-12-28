package deepseek

import (
	"context"
)

// ChatMessage represents a message in the conversation
type ChatMessage struct {
	Role    string `json:"role"` // "system", "user", or "assistant"
	Content string `json:"content"`
}

// ChatRequest represents a request to the DeepSeek API
type ChatRequest struct {
	Model       string        `json:"model"`
	Messages    []ChatMessage `json:"messages"`
	Temperature float64       `json:"temperature,omitempty"`
	MaxTokens   int           `json:"max_tokens,omitempty"`
	Stream      bool          `json:"stream,omitempty"`
}

// ChatChunk represents a streaming response chunk
type ChatChunk struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index        int     `json:"index"`
		Delta        Delta   `json:"delta"`
		FinishReason *string `json:"finish_reason"`
	} `json:"choices"`
}

// Delta represents the incremental content in a stream
type Delta struct {
	Role    string `json:"role,omitempty"`
	Content string `json:"content,omitempty"`
}

// ChatResponse represents a non-streaming response
type ChatResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index   int `json:"index"`
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

// Client interface for DeepSeek API interactions
type Client interface {
	// StreamChatCompletion sends a streaming chat completion request
	StreamChatCompletion(ctx context.Context, req ChatRequest) (<-chan ChatChunk, error)

	// ChatCompletion sends a non-streaming chat completion request (for memory extraction)
	ChatCompletion(ctx context.Context, req ChatRequest) (*ChatResponse, error)
}
