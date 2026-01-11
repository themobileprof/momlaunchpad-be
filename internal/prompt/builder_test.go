package prompt

import (
	"strings"
	"testing"
	"time"

	"github.com/themobileprof/momlaunchpad-be/internal/memory"
)

func TestBuilder_BuildPrompt(t *testing.T) {
	builder := NewBuilder()

	req := PromptRequest{
		UserID:      "user123",
		UserMessage: "When will my baby start kicking?",
		Language:    "en",
		ShortTermMemory: []memory.Message{
			{Role: "user", Content: "Hi, I'm 14 weeks pregnant", Timestamp: time.Now()},
			{Role: "assistant", Content: "Congratulations! How are you feeling?", Timestamp: time.Now()},
		},
		Facts: []memory.UserFact{
			{Key: "pregnancy_week", Value: "14", Confidence: 0.9},
			{Key: "diet", Value: "vegetarian", Confidence: 0.8},
		},
	}

	messages := builder.BuildPrompt(req)

	if len(messages) == 0 {
		t.Fatal("Expected messages, got empty slice")
	}

	if messages[0].Role != "system" {
		t.Errorf("Expected first message role 'system', got %q", messages[0].Role)
	}

	if !strings.Contains(messages[0].Content, "14") {
		t.Error("System prompt should include pregnancy week")
	}

	lastMsg := messages[len(messages)-1]
	if lastMsg.Role != "user" {
		t.Errorf("Expected last message role 'user', got %q", lastMsg.Role)
	}
	if lastMsg.Content != req.UserMessage {
		t.Errorf("Expected last message content %q, got %q", req.UserMessage, lastMsg.Content)
	}
}

func TestBuilder_BuildPromptSpanish(t *testing.T) {
	builder := NewBuilder()

	req := PromptRequest{
		UserID:          "user123",
		UserMessage:     "¿Cuándo empezará a moverse mi bebé?",
		Language:        "es",
		ShortTermMemory: []memory.Message{},
		Facts: []memory.UserFact{
			{Key: "pregnancy_week", Value: "16", Confidence: 0.9},
		},
	}

	messages := builder.BuildPrompt(req)

	if len(messages) == 0 {
		t.Fatal("Expected messages, got empty slice")
	}

	systemPrompt := messages[0].Content
	if !strings.Contains(systemPrompt, "Spanish") && !strings.Contains(systemPrompt, "Español") {
		t.Error("Expected system prompt to reference Spanish language")
	}
}

func TestBuilder_BuildPromptSmallTalk(t *testing.T) {
	builder := NewBuilder()

	req := PromptRequest{
		UserID:          "user123",
		UserMessage:     "hello",
		Language:        "en",
		IsSmallTalk:     true,
		ShortTermMemory: []memory.Message{},
		Facts:           []memory.UserFact{},
	}

	messages := builder.BuildPrompt(req)

	if len(messages) == 0 {
		return
	}

	if len(messages) > 3 {
		t.Errorf("Small talk should have minimal prompt, got %d messages", len(messages))
	}
}
func TestBuilder_AIName(t *testing.T) {
	builder := NewBuilder()

	tests := []struct {
		name         string
		aiName       string
		expectInText string
	}{
		{
			name:         "custom AI name",
			aiName:       "Luna",
			expectInText: "You are Luna",
		},
		{
			name:         "default when empty",
			aiName:       "",
			expectInText: "pregnancy support assistant",
		},
		{
			name:         "custom name MomBot",
			aiName:       "MomBot",
			expectInText: "You are MomBot",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := PromptRequest{
				UserID:      "user123",
				UserMessage: "When will my baby start kicking?",
				Language:    "en",
				AIName:      tt.aiName,
			}

			messages := builder.BuildPrompt(req)

			if len(messages) == 0 {
				t.Fatal("Expected messages, got empty slice")
			}

			systemPrompt := messages[0].Content
			if !strings.Contains(systemPrompt, tt.expectInText) {
				t.Errorf("System prompt should contain %q, got:\n%s", tt.expectInText, systemPrompt)
			}
		})
	}
}
