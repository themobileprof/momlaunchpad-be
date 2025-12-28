package deepseek

import (
	"context"
	"testing"
)

func TestMockClient_StreamChatCompletion(t *testing.T) {
	mock := NewMockClient()

	req := ChatRequest{
		Model: "deepseek-chat",
		Messages: []ChatMessage{
			{Role: "user", Content: "Hello"},
		},
		Stream: true,
	}

	ctx := context.Background()
	ch, err := mock.StreamChatCompletion(ctx, req)
	if err != nil {
		t.Fatalf("StreamChatCompletion() error = %v", err)
	}

	// Collect all chunks
	chunks := make([]ChatChunk, 0)
	for chunk := range ch {
		chunks = append(chunks, chunk)
	}

	if len(chunks) == 0 {
		t.Error("Expected at least one chunk, got none")
	}

	// Verify call was tracked
	if mock.GetStreamCallCount() != 1 {
		t.Errorf("Expected 1 stream call, got %d", mock.GetStreamCallCount())
	}
}

func TestMockClient_ChatCompletion(t *testing.T) {
	mock := NewMockClient()

	req := ChatRequest{
		Model: "deepseek-chat",
		Messages: []ChatMessage{
			{Role: "user", Content: "Extract facts"},
		},
		Stream: false,
	}

	ctx := context.Background()
	resp, err := mock.ChatCompletion(ctx, req)
	if err != nil {
		t.Fatalf("ChatCompletion() error = %v", err)
	}

	if resp == nil {
		t.Fatal("Expected response, got nil")
	}

	if len(resp.Choices) == 0 {
		t.Error("Expected at least one choice")
	}

	// Verify call was tracked
	if mock.GetChatCallCount() != 1 {
		t.Errorf("Expected 1 chat call, got %d", mock.GetChatCallCount())
	}
}

func TestMockClient_CustomStreamFunc(t *testing.T) {
	mock := NewMockClient()

	// Custom streaming behavior
	mock.StreamFunc = func(ctx context.Context, req ChatRequest) (<-chan ChatChunk, error) {
		ch := make(chan ChatChunk, 1)
		go func() {
			defer close(ch)
			ch <- ChatChunk{
				ID:    "custom-chunk",
				Model: "custom-model",
			}
		}()
		return ch, nil
	}

	ctx := context.Background()
	req := ChatRequest{Model: "test"}

	ch, err := mock.StreamChatCompletion(ctx, req)
	if err != nil {
		t.Fatalf("StreamChatCompletion() error = %v", err)
	}

	chunk := <-ch
	if chunk.ID != "custom-chunk" {
		t.Errorf("Expected custom-chunk, got %s", chunk.ID)
	}
}

func TestMockClient_Reset(t *testing.T) {
	mock := NewMockClient()

	ctx := context.Background()
	req := ChatRequest{Model: "test"}

	// Make some calls
	_, _ = mock.StreamChatCompletion(ctx, req)
	_, _ = mock.ChatCompletion(ctx, req)

	if mock.GetStreamCallCount() != 1 || mock.GetChatCallCount() != 1 {
		t.Error("Calls not tracked before reset")
	}

	// Reset
	mock.Reset()

	if mock.GetStreamCallCount() != 0 || mock.GetChatCallCount() != 0 {
		t.Error("Reset did not clear call history")
	}
}
