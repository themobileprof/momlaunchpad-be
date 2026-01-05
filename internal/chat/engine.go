package chat

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/themobileprof/momlaunchpad-be/internal/calendar"
	"github.com/themobileprof/momlaunchpad-be/internal/circuitbreaker"
	"github.com/themobileprof/momlaunchpad-be/internal/classifier"
	"github.com/themobileprof/momlaunchpad-be/internal/db"
	"github.com/themobileprof/momlaunchpad-be/internal/fallback"
	"github.com/themobileprof/momlaunchpad-be/internal/language"
	"github.com/themobileprof/momlaunchpad-be/internal/memory"
	"github.com/themobileprof/momlaunchpad-be/internal/privacy"
	"github.com/themobileprof/momlaunchpad-be/internal/prompt"
	"github.com/themobileprof/momlaunchpad-be/pkg/deepseek"
)

// Responder defines the interface for sending responses to any transport
type Responder interface {
	SendMessage(content string) error
	SendCalendarSuggestion(suggestion calendar.Suggestion) error
	SendError(message string) error
	SendDone() error
}

// ProcessRequest contains all data needed to process a message
type ProcessRequest struct {
	UserID    string
	Message   string
	Language  string
	Responder Responder
}

// Engine handles core conversation logic independent of transport
type Engine struct {
	classifier     ClassifierInterface
	memoryManager  MemoryInterface
	promptBuilder  PromptInterface
	deepseekClient deepseek.Client
	calSuggester   CalendarInterface
	langManager    LanguageInterface
	db             DBInterface
	circuitBreaker *circuitbreaker.CircuitBreaker
	aiTimeout      time.Duration
}

// Interfaces for dependencies
type ClassifierInterface interface {
	Classify(text, language string) classifier.ClassifierResult
}

type MemoryInterface interface {
	AddMessage(userID string, msg memory.Message)
	GetShortTermMemory(userID string) []memory.Message
}

type PromptInterface interface {
	BuildPrompt(req prompt.PromptRequest) []deepseek.ChatMessage
}

type CalendarInterface interface {
	ShouldSuggest(intent classifier.Intent, message string) calendar.SuggestionResult
	BuildSuggestion(intent classifier.Intent, message string) calendar.Suggestion
}

type LanguageInterface interface {
	Validate(code string) language.ValidationResult
}

type DBInterface interface {
	SaveMessage(ctx context.Context, userID, role, content string) (*db.Message, error)
	GetUserFacts(ctx context.Context, userID string) ([]db.UserFact, error)
	SaveOrUpdateFact(ctx context.Context, userID, key, value string, confidence float64) (*db.UserFact, error)
}

// NewEngine creates a new transport-agnostic chat engine
func NewEngine(
	cls ClassifierInterface,
	mem MemoryInterface,
	pb PromptInterface,
	ds deepseek.Client,
	cal CalendarInterface,
	lm LanguageInterface,
	database DBInterface,
) *Engine {
	return &Engine{
		classifier:     cls,
		memoryManager:  mem,
		promptBuilder:  pb,
		deepseekClient: ds,
		calSuggester:   cal,
		langManager:    lm,
		db:             database,
		circuitBreaker: circuitbreaker.NewCircuitBreaker(5, 5*time.Minute),
		aiTimeout:      30 * time.Second,
	}
}

// ProcessMessage processes a chat message and sends responses via the provided responder
func (e *Engine) ProcessMessage(ctx context.Context, req ProcessRequest) error {
	log.Printf("Processing message: userID=%s, length=%d", req.UserID, len(req.Message))

	if privacy.ContainsPII(req.Message) {
		log.Printf("Warning: Potential PII detected in message from user=%s", req.UserID)
	}

	result := e.classifier.Classify(req.Message, req.Language)
	log.Printf("Intent classified: %s (confidence: %.2f)", result.Intent, result.Confidence)

	if _, err := e.db.SaveMessage(ctx, req.UserID, "user", req.Message); err != nil {
		return fmt.Errorf("failed to save message: %w", err)
	}

	e.memoryManager.AddMessage(req.UserID, memory.Message{
		Role:    "user",
		Content: req.Message,
	})

	if result.Intent == classifier.IntentSmallTalk {
		response := getSmallTalkResponse(req.Language)
		if err := req.Responder.SendMessage(response); err != nil {
			return err
		}
		return req.Responder.SendDone()
	}

	if shouldSuggest := e.calSuggester.ShouldSuggest(result.Intent, req.Message); shouldSuggest.ShouldSuggest {
		suggestion := e.calSuggester.BuildSuggestion(result.Intent, req.Message)
		if err := req.Responder.SendCalendarSuggestion(suggestion); err != nil {
			return err
		}
	}

	if e.circuitBreaker.State() == circuitbreaker.StateOpen {
		log.Printf("Circuit breaker open, using fallback response")
		fbResp := fallback.GetCircuitOpenResponse(req.Language)
		req.Responder.SendMessage(fbResp.Content)
		return req.Responder.SendDone()
	}

	facts, _ := e.db.GetUserFacts(ctx, req.UserID)
	shortTermMsgs := e.memoryManager.GetShortTermMemory(req.UserID)
	sanitizedContent := privacy.SanitizeForAPI(req.Message)

	promptReq := prompt.PromptRequest{
		UserID:          req.UserID,
		UserMessage:     sanitizedContent,
		Language:        req.Language,
		IsSmallTalk:     result.Intent == classifier.IntentSmallTalk,
		ShortTermMemory: shortTermMsgs,
		Facts:           convertDBFactsToMemoryFacts(facts),
	}
	messages := e.promptBuilder.BuildPrompt(promptReq)

	ctxWithTimeout, cancel := context.WithTimeout(ctx, e.aiTimeout)
	defer cancel()

	deepseekReq := deepseek.ChatRequest{
		Model:       "deepseek-chat",
		Messages:    messages,
		Temperature: 0.7,
		MaxTokens:   1000,
		Stream:      true,
	}

	var fullResponse strings.Builder
	err := e.circuitBreaker.Call(func() error {
		chunks, err := e.deepseekClient.StreamChatCompletion(ctxWithTimeout, deepseekReq)
		if err != nil {
			return err
		}

		for chunk := range chunks {
			select {
			case <-ctxWithTimeout.Done():
				return context.DeadlineExceeded
			default:
			}

			if len(chunk.Choices) == 0 {
				log.Printf("Warning: Empty chunk received from DeepSeek")
				continue
			}

			chunkContent := chunk.Choices[0].Delta.Content
			if chunkContent != "" {
				fullResponse.WriteString(chunkContent)
				if err := req.Responder.SendMessage(chunkContent); err != nil {
					return err
				}
			}
		}
		return nil
	})

	if err != nil {
		log.Printf("AI call failed: %v", err)

		var fbResp fallback.Response
		if errors.Is(err, context.DeadlineExceeded) {
			fbResp = fallback.GetTimeoutResponse(req.Language)
		} else {
			fbResp = fallback.GetFallbackResponse(result.Intent, req.Language)
		}

		req.Responder.SendMessage(fbResp.Content)
		return req.Responder.SendDone()
	}

	assistantMsg := fullResponse.String()
	if _, err := e.db.SaveMessage(ctx, req.UserID, "assistant", assistantMsg); err != nil {
		log.Printf("Failed to save assistant message: %v", err)
	}
	e.memoryManager.AddMessage(req.UserID, memory.Message{
		Role:    "assistant",
		Content: assistantMsg,
	})

	e.extractAndSaveFacts(ctx, req.UserID, req.Message, assistantMsg)

	return req.Responder.SendDone()
}

func getSmallTalkResponse(language string) string {
	responses := map[string]string{
		"en": "I'm here with you. How can I help today?",
		"es": "Estoy aquí contigo. ¿Cómo puedo ayudarte hoy?",
	}

	if resp, ok := responses[language]; ok {
		return resp
	}
	return responses["en"]
}

func convertDBFactsToMemoryFacts(dbFacts []db.UserFact) []memory.UserFact {
	memFacts := make([]memory.UserFact, len(dbFacts))
	for i, f := range dbFacts {
		memFacts[i] = memory.UserFact{
			Key:        f.Key,
			Value:      f.Value,
			Confidence: f.Confidence,
			UpdatedAt:  f.UpdatedAt,
		}
	}
	return memFacts
}

func (e *Engine) extractAndSaveFacts(ctx context.Context, userID, userMsg, aiMsg string) {
	normalized := strings.ToLower(userMsg)

	if strings.Contains(normalized, "week") && strings.Contains(normalized, "pregnant") {
		for i := 1; i <= 42; i++ {
			weekStr := fmt.Sprintf("%d week", i)
			if strings.Contains(normalized, weekStr) {
				e.db.SaveOrUpdateFact(ctx, userID, "pregnancy_week", fmt.Sprintf("%d", i), 0.8)
				break
			}
		}
	}
}
