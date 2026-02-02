package llm

import (
	"context"
)

// Client interface for LLM API interactions
type Client interface {
	// StreamChatCompletion sends a streaming chat completion request
	StreamChatCompletion(ctx context.Context, req ChatRequest) (<-chan ChatChunk, error)

	// ChatCompletion sends a non-streaming chat completion request
	ChatCompletion(ctx context.Context, req ChatRequest) (*ChatResponse, error)
}
