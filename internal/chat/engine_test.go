package chat

import (
	"context"
	"testing"

	"github.com/themobileprof/momlaunchpad-be/internal/calendar"
	"github.com/themobileprof/momlaunchpad-be/internal/classifier"
	"github.com/themobileprof/momlaunchpad-be/internal/db"
	"github.com/themobileprof/momlaunchpad-be/internal/language"
	"github.com/themobileprof/momlaunchpad-be/internal/memory"
	"github.com/themobileprof/momlaunchpad-be/internal/prompt"
	"github.com/themobileprof/momlaunchpad-be/pkg/deepseek"
)

func TestEngineCreation(t *testing.T) {
	engine := NewEngine(
		&mockClassifier{},
		&mockMemoryManager{},
		&mockPromptBuilder{},
		&mockDeepSeekClient{},
		&mockCalSuggester{},
		&mockLangManager{},
		&mockDB{},
	)
	if engine == nil {
		t.Fatal("expected engine to be created")
	}
}

type mockClassifier struct{}

func (m *mockClassifier) Classify(text, language string) classifier.ClassifierResult {
	return classifier.ClassifierResult{Intent: classifier.IntentSmallTalk, Confidence: 0.9}
}

type mockMemoryManager struct{ messages []memory.Message }

func (m *mockMemoryManager) AddMessage(userID string, msg memory.Message) {
	m.messages = append(m.messages, msg)
}
func (m *mockMemoryManager) GetShortTermMemory(userID string) []memory.Message {
	return m.messages
}

type mockPromptBuilder struct{}

func (m *mockPromptBuilder) BuildPrompt(req prompt.PromptRequest) []deepseek.ChatMessage {
	return []deepseek.ChatMessage{
		{Role: "system", Content: "test"},
		{Role: "user", Content: req.UserMessage},
	}
}

type mockDeepSeekClient struct{}

func (m *mockDeepSeekClient) StreamChatCompletion(ctx context.Context, req deepseek.ChatRequest) (<-chan deepseek.ChatChunk, error) {
	ch := make(chan deepseek.ChatChunk, 1)
	go func() {
		defer close(ch)
		chunk := deepseek.ChatChunk{}
		chunk.Choices = make([]struct {
			Index        int            `json:"index"`
			Delta        deepseek.Delta `json:"delta"`
			FinishReason *string        `json:"finish_reason"`
		}, 1)
		chunk.Choices[0].Delta.Content = "Test response"
		ch <- chunk
		close(ch)
	}()
	return ch, nil
}
func (m *mockDeepSeekClient) ChatCompletion(ctx context.Context, req deepseek.ChatRequest) (*deepseek.ChatResponse, error) {
	return &deepseek.ChatResponse{}, nil
}

type mockCalSuggester struct{}

func (m *mockCalSuggester) ShouldSuggest(intent classifier.Intent, message string) calendar.SuggestionResult {
	return calendar.SuggestionResult{ShouldSuggest: false}
}
func (m *mockCalSuggester) BuildSuggestion(intent classifier.Intent, message string) calendar.Suggestion {
	return calendar.Suggestion{Type: "appointment", Title: "Test", Description: "Test"}
}

type mockLangManager struct{}

func (m *mockLangManager) Validate(code string) language.ValidationResult {
	return language.ValidationResult{Code: code, UsedFallback: false}
}

type mockDB struct {
	messages []string
	facts    []string
}

func (m *mockDB) SaveMessage(ctx context.Context, userID, role, content string) (*db.Message, error) {
	m.messages = append(m.messages, userID+":"+role+":"+content)
	return &db.Message{}, nil
}
func (m *mockDB) GetUserFacts(ctx context.Context, userID string) ([]db.UserFact, error) {
	return []db.UserFact{}, nil
}
func (m *mockDB) SaveOrUpdateFact(ctx context.Context, userID, key, value string, confidence float64) (*db.UserFact, error) {
	m.facts = append(m.facts, key+":"+value)
	return &db.UserFact{}, nil
}
func (m *mockDB) SaveSymptom(ctx context.Context, userID, symptomType, description, severity, frequency, onsetTime string, associatedSymptoms []string) (string, error) {
	return "mock-symptom-id", nil
}
func (m *mockDB) GetRecentSymptoms(ctx context.Context, userID string, limit int) ([]map[string]interface{}, error) {
	return []map[string]interface{}{}, nil
}
func (m *mockDB) GetSystemSetting(ctx context.Context, key string) (*db.SystemSetting, error) {
	if key == "ai_name" {
		return &db.SystemSetting{Key: "ai_name", Value: "MomBot"}, nil
	}
	return nil, db.ErrNotFound
}

type mockResponder struct {
	messages []string
	done     bool
}

func (m *mockResponder) SendMessage(content string) error {
	m.messages = append(m.messages, content)
	return nil
}
func (m *mockResponder) SendCalendarSuggestion(suggestion calendar.Suggestion) error { return nil }
func (m *mockResponder) SendError(message string) error                              { return nil }
func (m *mockResponder) SendDone() error                                             { m.done = true; return nil }
