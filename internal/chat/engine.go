package chat

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/themobileprof/momlaunchpad-be/internal/calendar"
	"github.com/themobileprof/momlaunchpad-be/internal/circuitbreaker"
	"github.com/themobileprof/momlaunchpad-be/internal/classifier"
	"github.com/themobileprof/momlaunchpad-be/internal/conversation"
	"github.com/themobileprof/momlaunchpad-be/internal/db"
	"github.com/themobileprof/momlaunchpad-be/internal/fallback"
	"github.com/themobileprof/momlaunchpad-be/internal/language"
	"github.com/themobileprof/momlaunchpad-be/internal/memory"
	"github.com/themobileprof/momlaunchpad-be/internal/privacy"
	"github.com/themobileprof/momlaunchpad-be/internal/prompt"
	"github.com/themobileprof/momlaunchpad-be/internal/symptoms"
	"github.com/themobileprof/momlaunchpad-be/pkg/llm"
)

// Responder defines the interface for sending responses to any transport
type Responder interface {
	SendMessage(content string) error
	SendCalendarSuggestion(suggestion calendar.Suggestion) error
	SendError(message string) error
	SendDone() error
	SetConversationID(id string)
}

// ProcessRequest contains all data needed to process a message
type ProcessRequest struct {
	UserID         string
	ConversationID string
	Message        string
	Language       string
	Responder      Responder
}

// Engine handles core conversation logic independent of transport
type Engine struct {
	classifier     ClassifierInterface
	memoryManager  MemoryInterface
	promptBuilder  PromptInterface
	llmClient      llm.Client
	calSuggester   CalendarInterface
	langManager    LanguageInterface
	db             DBInterface
	convManager    *conversation.Manager
	symptomTracker *symptoms.Tracker
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
	BuildPrompt(req prompt.PromptRequest) []llm.ChatMessage
}

type CalendarInterface interface {
	ShouldSuggest(intent classifier.Intent, message string) calendar.SuggestionResult
	BuildSuggestion(intent classifier.Intent, message string) calendar.Suggestion
}

type LanguageInterface interface {
	Validate(code string) language.ValidationResult
}

type DBInterface interface {
	SaveMessage(ctx context.Context, userID, conversationID, role, content string) (*db.Message, error)
	CreateConversation(ctx context.Context, userID string, title *string) (*db.Conversation, error)
	GetUserFacts(ctx context.Context, userID string) ([]db.UserFact, error)
	SaveSymptom(ctx context.Context, userID, symptomType, description, severity, frequency, onsetTime string, associatedSymptoms []string) (string, error)
	GetRecentSymptoms(ctx context.Context, userID string, limit int) ([]map[string]interface{}, error)
	SaveOrUpdateFact(ctx context.Context, userID, key, value string, confidence float64) (*db.UserFact, error)
	GetSystemSetting(ctx context.Context, key string) (*db.SystemSetting, error)
}

// NewEngine creates a new transport-agnostic chat engine
func NewEngine(
	cls ClassifierInterface,
	mem MemoryInterface,
	pb PromptInterface,
	client llm.Client,
	cal CalendarInterface,
	lm LanguageInterface,
	database DBInterface,
) *Engine {
	return &Engine{
		classifier:     cls,
		memoryManager:  mem,
		promptBuilder:  pb,
		llmClient:      client,
		calSuggester:   cal,
		langManager:    lm,
		db:             database,
		convManager:    conversation.NewManager(),
		symptomTracker: symptoms.NewTracker(),
		circuitBreaker: circuitbreaker.NewCircuitBreaker(5, 5*time.Minute),
		aiTimeout:      30 * time.Second,
	}
}

// ProcessMessage processes a chat message and sends responses via the provided responder
func (e *Engine) ProcessMessage(ctx context.Context, req ProcessRequest) (string, error) {
	log.Printf("Processing message: userID=%s, length=%d", req.UserID, len(req.Message))

	// Ensure conversation ID exists
	conversationID := req.ConversationID
	if conversationID == "" {
		// Auto-generate title from first few words (up to 50 chars)
		title := req.Message
		if len(title) > 50 {
			title = title[:47] + "..."
		}
		
		newConv, err := e.db.CreateConversation(ctx, req.UserID, &title)
		if err != nil {
			return "", fmt.Errorf("failed to create conversation: %w", err)
		}
		conversationID = newConv.ID
		log.Printf("Created new conversation: %s", conversationID)
		
		// Optionally notify responder of new conversation ID? 
		// For now, we assume the client will see it in the message history or list query.
	}
	
	// Notify responder of conversation ID
	req.Responder.SetConversationID(conversationID)

	if privacy.ContainsPII(req.Message) {
		log.Printf("Warning: Potential PII detected in message from user=%s", req.UserID)
	}

	result := e.classifier.Classify(req.Message, req.Language)
	log.Printf("Intent classified: %s (confidence: %.2f)", result.Intent, result.Confidence)

	if _, err := e.db.SaveMessage(ctx, req.UserID, conversationID, "user", req.Message); err != nil {
		return conversationID, fmt.Errorf("failed to save message: %w", err)
	}

	e.memoryManager.AddMessage(req.UserID, memory.Message{
		Role:    "user",
		Content: req.Message,
	})

	if result.Intent == classifier.IntentSmallTalk {
		response := getSmallTalkResponse(req.Language)
		if err := req.Responder.SendMessage(response); err != nil {
			return conversationID, err
		}
		
		// Also save assistant response
		if _, err := e.db.SaveMessage(ctx, req.UserID, conversationID, "assistant", response); err != nil {
			log.Printf("Failed to save assistant message: %v", err)
		}
		
		// Reset conversation state on small talk
		e.convManager.Reset(req.UserID)
		return conversationID, req.Responder.SendDone()
	}

	// Get conversation state
	convState := e.convManager.GetState(req.UserID)

	// Extract and save symptoms if present (for symptom reports or pregnancy questions)
	if result.Intent == classifier.IntentSymptom || result.Intent == classifier.IntentPregnancyQ {
		extractedSymptoms := e.symptomTracker.ExtractSymptoms(req.Message)
		if len(extractedSymptoms) > 0 {
			log.Printf("Extracted %d symptom(s) from message", len(extractedSymptoms))
			for _, symptom := range extractedSymptoms {
				symptomID, err := e.db.SaveSymptom(
					ctx,
					req.UserID,
					symptom.Type,
					symptom.Description,
					symptom.Severity,
					symptom.Frequency,
					symptom.OnsetTime,
					symptom.AssociatedSymptoms,
				)
				if err != nil {
					log.Printf("Warning: failed to save symptom: %v", err)
				} else {
					log.Printf("Saved symptom: %s (ID: %s)", symptom.Type, symptomID)
				}
			}
		}
	}

	// Detect primary concern from first substantive message
	if convState.PrimaryConcern == "" {
		primaryConcern := extractPrimaryConcern(req.Message)
		e.convManager.SetPrimaryConcern(req.UserID, primaryConcern)
		log.Printf("Set primary concern for user %s: %s", req.UserID, primaryConcern)
	} else {
		// Track if user mentioned new topics
		if containsNewSymptom(req.Message, convState.PrimaryConcern) {
			e.convManager.AddSecondaryTopic(req.UserID, req.Message)
			log.Printf("Detected secondary topic for user %s", req.UserID)
		}

		// Increment follow-up count
		e.convManager.IncrementFollowUp(req.UserID)
	}

	if shouldSuggest := e.calSuggester.ShouldSuggest(result.Intent, req.Message); shouldSuggest.ShouldSuggest {
		suggestion := e.calSuggester.BuildSuggestion(result.Intent, req.Message)
		if err := req.Responder.SendCalendarSuggestion(suggestion); err != nil {
			return conversationID, err
		}
	}

	if e.circuitBreaker.State() == circuitbreaker.StateOpen {
		log.Printf("Circuit breaker open, using fallback response")
		fbResp := fallback.GetCircuitOpenResponse(req.Language)
		req.Responder.SendMessage(fbResp.Content)
		return conversationID, req.Responder.SendDone()
	}

	// Fetch facts, symptoms, AI name, and short-term memory concurrently for speed
	var (
		facts          []db.UserFact
		recentSymptoms []map[string]interface{}
		shortTermMsgs  []memory.Message
		aiName         string
		wg             sync.WaitGroup
	)
	wg.Add(4)
	go func() {
		defer wg.Done()
		facts, _ = e.db.GetUserFacts(ctx, req.UserID)
	}()
	go func() {
		defer wg.Done()
		recentSymptoms, _ = e.db.GetRecentSymptoms(ctx, req.UserID, 10) // Last 10 symptoms
	}()
	go func() {
		defer wg.Done()
		shortTermMsgs = e.memoryManager.GetShortTermMemory(req.UserID)
	}()
	go func() {
		defer wg.Done()
		// Fetch AI name from system settings
		setting, err := e.db.GetSystemSetting(ctx, "ai_name")
		if err == nil && setting != nil {
			aiName = setting.Value
		} else {
			aiName = "MomBot" // Fallback default
		}
	}()
	wg.Wait()

	sanitizedContent := privacy.SanitizeForAPI(req.Message)

	log.Printf("Building prompt for user=%s, intent=%s, aiName=%s", req.UserID, result.Intent, aiName)

	promptReq := prompt.PromptRequest{
		UserID:            req.UserID,
		UserMessage:       sanitizedContent,
		Language:          req.Language,
		IsSmallTalk:       result.Intent == classifier.IntentSmallTalk,
		ShortTermMemory:   shortTermMsgs,
		Facts:             convertDBFactsToMemoryFacts(facts),
		RecentSymptoms:    recentSymptoms,
		ConversationState: convState,
		AIName:            aiName, // Pass AI name to prompt builder
	}
	messages := e.promptBuilder.BuildPrompt(promptReq)

	log.Printf("Calling LLM API with %d messages, maxTokens=%d", len(messages), 200)

	ctxWithTimeout, cancel := context.WithTimeout(ctx, e.aiTimeout)
	defer cancel()

	chatReq := llm.ChatRequest{
		Model:       "",    // Let client use default or we can inject it
		Messages:    messages,
		Temperature: 0.7,
		MaxTokens:   200,   // Limit to ~150 words for voice-friendly, concise responses
		Stream:      false, // Non-streaming for short responses
	}

	log.Printf("Calling LLM API for user=%s", req.UserID)
	var assistantMsg string
	err := e.circuitBreaker.Call(func() error {
		response, err := e.llmClient.ChatCompletion(ctxWithTimeout, chatReq)
		if err != nil {
			log.Printf("LLM API error: %v", err)
			return err
		}

		if len(response.Choices) == 0 {
			return fmt.Errorf("no response from LLM")
		}

		assistantMsg = response.Choices[0].Message.Content
		log.Printf("AI response received: %d bytes", len(assistantMsg))
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
		return conversationID, req.Responder.SendDone()
	}

	// Send the complete response at once
	if err := req.Responder.SendMessage(assistantMsg); err != nil {
		return conversationID, fmt.Errorf("failed to send message: %w", err)
	}

	// Save assistant message to DB and memory
	if _, err := e.db.SaveMessage(ctx, req.UserID, conversationID, "assistant", assistantMsg); err != nil {
		log.Printf("Failed to save assistant message: %v", err)
	}
	e.memoryManager.AddMessage(req.UserID, memory.Message{
		Role:    "assistant",
		Content: assistantMsg,
	})

	e.extractAndSaveFacts(ctx, req.UserID, req.Message, assistantMsg)

	return conversationID, req.Responder.SendDone()
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

// extractPrimaryConcern extracts the main symptom/concern from message
func extractPrimaryConcern(message string) string {
	lower := strings.ToLower(message)

	// Common pregnancy symptoms - return first match
	symptoms := map[string]string{
		"swollen":   "swollen feet/ankles",
		"swell":     "swelling",
		"nausea":    "nausea/morning sickness",
		"headache":  "headaches",
		"back pain": "back pain",
		"cramp":     "cramping",
		"blurry":    "vision changes",
		"vision":    "vision changes",
		"dizzy":     "dizziness",
		"tired":     "fatigue",
		"insomnia":  "sleep issues",
		"heartburn": "heartburn",
		"vomit":     "vomiting",
		"constipa":  "constipation",
		"bleed":     "bleeding",
	}

	for keyword, concern := range symptoms {
		if strings.Contains(lower, keyword) {
			return concern
		}
	}

	// Default: use first few words
	words := strings.Fields(message)
	if len(words) > 5 {
		return strings.Join(words[:5], " ")
	}
	return message
}

// containsNewSymptom checks if message mentions a different symptom
func containsNewSymptom(message, primaryConcern string) bool {
	lower := strings.ToLower(message)
	concernLower := strings.ToLower(primaryConcern)

	// If message contains primary concern keywords, it's not new
	concernWords := strings.Fields(concernLower)
	matchCount := 0
	for _, word := range concernWords {
		if len(word) > 3 && strings.Contains(lower, word) {
			matchCount++
		}
	}

	// If less than half the concern words match, likely a new topic
	return matchCount < len(concernWords)/2
}
