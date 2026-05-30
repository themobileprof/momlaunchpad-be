package welcome

import (
	"context"
	"strings"
	"testing"

	"github.com/themobileprof/momlaunchpad-be/internal/db"
	"github.com/themobileprof/momlaunchpad-be/pkg/llm"
)

type mockLLM struct {
	response string
	err      error
	called   int
}

func (m *mockLLM) StreamChatCompletion(_ context.Context, _ llm.ChatRequest) (<-chan llm.ChatChunk, error) {
	return nil, nil
}

func (m *mockLLM) ChatCompletion(_ context.Context, _ llm.ChatRequest) (*llm.ChatResponse, error) {
	m.called++
	if m.err != nil {
		return nil, m.err
	}
	return &llm.ChatResponse{
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
					Content: m.response,
				},
			},
		},
	}, nil
}

func TestGenerateMessageFallsBackToDeepseek(t *testing.T) {
	name := "Sarah"
	user := &db.User{Name: &name}

	gemini := &mockLLM{err: context.DeadlineExceeded}
	deepseek := &mockLLM{response: "Hi Sarah! DeepSeek welcome."}

	svc := NewService(nil, gemini, deepseek)
	message, source, err := svc.generateMessage(context.Background(), user, "No detailed health records yet.")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if source != "deepseek" {
		t.Fatalf("expected deepseek source, got %q", source)
	}
	if message != "Hi Sarah! DeepSeek welcome." {
		t.Fatalf("unexpected message: %q", message)
	}
	if gemini.called != 1 || deepseek.called != 1 {
		t.Fatalf("expected one call each, gemini=%d deepseek=%d", gemini.called, deepseek.called)
	}
}

func TestGenerateMessageUsesGeminiWhenAvailable(t *testing.T) {
	name := "Sarah"
	user := &db.User{Name: &name}

	gemini := &mockLLM{response: "Hi Sarah! Gemini welcome."}
	deepseek := &mockLLM{response: "should not be used"}

	svc := NewService(nil, gemini, deepseek)
	message, source, err := svc.generateMessage(context.Background(), user, "context")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if source != "gemini" {
		t.Fatalf("expected gemini source, got %q", source)
	}
	if message != "Hi Sarah! Gemini welcome." {
		t.Fatalf("unexpected message: %q", message)
	}
	if deepseek.called != 0 {
		t.Fatalf("deepseek should not be called when gemini succeeds")
	}
}

func TestFallbackWelcomeUsesFirstName(t *testing.T) {
	name := "Sarah Johnson"
	week := 32
	user := &db.User{
		Name:          &name,
		PregnancyWeek: &week,
	}

	msg := fallbackWelcome(user)
	if !strings.Contains(msg, "Sarah") {
		t.Fatalf("expected name in fallback, got: %s", msg)
	}
	if !strings.Contains(msg, "Week 32") {
		t.Fatalf("expected week in fallback, got: %s", msg)
	}
}

func TestDisplayFirstName(t *testing.T) {
	name := "Sarah"
	user := &db.User{Name: &name}
	if got := displayFirstName(user); got != "Sarah" {
		t.Fatalf("got %q", got)
	}
}
