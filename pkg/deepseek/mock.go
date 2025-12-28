package deepseek

import (
	"context"
	"sync"
)

// MockClient implements the Client interface for testing
type MockClient struct {
	mu sync.Mutex

	// StreamFunc allows customizing the streaming behavior
	StreamFunc func(context.Context, ChatRequest) (<-chan ChatChunk, error)

	// ChatFunc allows customizing the non-streaming behavior
	ChatFunc func(context.Context, ChatRequest) (*ChatResponse, error)

	// Tracking for assertions
	StreamCalls []ChatRequest
	ChatCalls   []ChatRequest
}

// NewMockClient creates a new mock client with default behavior
func NewMockClient() *MockClient {
	return &MockClient{
		StreamCalls: make([]ChatRequest, 0),
		ChatCalls:   make([]ChatRequest, 0),
	}
}

// StreamChatCompletion implements Client.StreamChatCompletion
func (m *MockClient) StreamChatCompletion(ctx context.Context, req ChatRequest) (<-chan ChatChunk, error) {
	m.mu.Lock()
	m.StreamCalls = append(m.StreamCalls, req)
	m.mu.Unlock()

	if m.StreamFunc != nil {
		return m.StreamFunc(ctx, req)
	}

	// Default mock behavior - return a simple response
	ch := make(chan ChatChunk, 3)
	go func() {
		defer close(ch)

		ch <- ChatChunk{
			ID:      "mock-chunk-1",
			Object:  "chat.completion.chunk",
			Created: 1234567890,
			Model:   req.Model,
			Choices: []struct {
				Index        int     `json:"index"`
				Delta        Delta   `json:"delta"`
				FinishReason *string `json:"finish_reason"`
			}{
				{
					Index: 0,
					Delta: Delta{
						Content: "This is ",
					},
				},
			},
		}

		ch <- ChatChunk{
			ID:      "mock-chunk-2",
			Object:  "chat.completion.chunk",
			Created: 1234567890,
			Model:   req.Model,
			Choices: []struct {
				Index        int     `json:"index"`
				Delta        Delta   `json:"delta"`
				FinishReason *string `json:"finish_reason"`
			}{
				{
					Index: 0,
					Delta: Delta{
						Content: "a mock response.",
					},
				},
			},
		}

		finishReason := "stop"
		ch <- ChatChunk{
			ID:      "mock-chunk-3",
			Object:  "chat.completion.chunk",
			Created: 1234567890,
			Model:   req.Model,
			Choices: []struct {
				Index        int     `json:"index"`
				Delta        Delta   `json:"delta"`
				FinishReason *string `json:"finish_reason"`
			}{
				{
					Index:        0,
					Delta:        Delta{},
					FinishReason: &finishReason,
				},
			},
		}
	}()

	return ch, nil
}

// ChatCompletion implements Client.ChatCompletion
func (m *MockClient) ChatCompletion(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	m.mu.Lock()
	m.ChatCalls = append(m.ChatCalls, req)
	m.mu.Unlock()

	if m.ChatFunc != nil {
		return m.ChatFunc(ctx, req)
	}

	// Default mock behavior
	return &ChatResponse{
		ID:      "mock-response-1",
		Object:  "chat.completion",
		Created: 1234567890,
		Model:   req.Model,
		Choices: []struct {
			Index   int `json:"index"`
			Message struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			} `json:"message"`
			FinishReason string `json:"finish_reason"`
		}{
			{
				Index: 0,
				Message: struct {
					Role    string `json:"role"`
					Content string `json:"content"`
				}{
					Role:    "assistant",
					Content: "This is a mock response.",
				},
				FinishReason: "stop",
			},
		},
		Usage: struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
			TotalTokens      int `json:"total_tokens"`
		}{
			PromptTokens:     10,
			CompletionTokens: 5,
			TotalTokens:      15,
		},
	}, nil
}

// Reset clears the call history
func (m *MockClient) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.StreamCalls = make([]ChatRequest, 0)
	m.ChatCalls = make([]ChatRequest, 0)
}

// GetStreamCallCount returns the number of stream calls made
func (m *MockClient) GetStreamCallCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.StreamCalls)
}

// GetChatCallCount returns the number of chat calls made
func (m *MockClient) GetChatCallCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.ChatCalls)
}
