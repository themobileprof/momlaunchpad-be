package ws

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/websocket"
	"github.com/themobileprof/momlaunchpad-be/internal/api/middleware"
	"github.com/themobileprof/momlaunchpad-be/internal/calendar"
	"github.com/themobileprof/momlaunchpad-be/internal/classifier"
	"github.com/themobileprof/momlaunchpad-be/internal/db"
	"github.com/themobileprof/momlaunchpad-be/internal/language"
	"github.com/themobileprof/momlaunchpad-be/internal/memory"
	"github.com/themobileprof/momlaunchpad-be/internal/prompt"
	"github.com/themobileprof/momlaunchpad-be/pkg/deepseek"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for now
	},
}

// ChatHandler handles WebSocket chat connections
type ChatHandler struct {
	classifier     *classifier.Classifier
	memoryManager  *memory.MemoryManager
	promptBuilder  *prompt.Builder
	deepseekClient deepseek.Client
	calSuggester   *calendar.Suggester
	langManager    *language.Manager
	db             *db.DB
	jwtSecret      string
}

// NewChatHandler creates a new chat handler
func NewChatHandler(
	cls *classifier.Classifier,
	mem *memory.MemoryManager,
	pb *prompt.Builder,
	ds deepseek.Client,
	cal *calendar.Suggester,
	lm *language.Manager,
	database *db.DB,
	jwtSecret string,
) *ChatHandler {
	return &ChatHandler{
		classifier:     cls,
		memoryManager:  mem,
		promptBuilder:  pb,
		deepseekClient: ds,
		calSuggester:   cal,
		langManager:    lm,
		db:             database,
		jwtSecret:      jwtSecret,
	}
}

// IncomingMessage represents a message from the client
type IncomingMessage struct {
	Content string `json:"content"`
}

// OutgoingMessage represents a message to the client
type OutgoingMessage struct {
	Type    string      `json:"type"` // "message", "calendar", "error", "done"
	Content string      `json:"content,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

// HandleChat handles WebSocket chat connections
func (h *ChatHandler) HandleChat(c *gin.Context) {
	// Validate JWT from query parameter or header
	token := c.Query("token")
	if token == "" {
		token = c.GetHeader("Authorization")
		if strings.HasPrefix(token, "Bearer ") {
			token = strings.TrimPrefix(token, "Bearer ")
		}
	}

	if token == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Missing token"})
		return
	}

	// Parse JWT
	claims := &middleware.JWTClaims{}
	jwtToken, err := jwt.ParseWithClaims(token, claims, func(t *jwt.Token) (interface{}, error) {
		return []byte(h.jwtSecret), nil
	})

	if err != nil || !jwtToken.Valid {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
		return
	}

	// Upgrade to WebSocket
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}
	defer conn.Close()

	userID := claims.UserID

	// Get user from database to get language preference
	user, err := h.db.GetUserByID(c.Request.Context(), userID)
	if err != nil {
		log.Printf("Failed to get user: %v", err)
		return
	}

	// Validate language
	langResult := h.langManager.Validate(user.Language)
	userLanguage := langResult.Code

	log.Printf("WebSocket connected: user=%s, language=%s", userID, userLanguage)

	// Listen for messages
	for {
		var msg IncomingMessage
		if err := conn.ReadJSON(&msg); err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}

		if err := h.processMessage(c.Request.Context(), conn, userID, userLanguage, msg.Content); err != nil {
			log.Printf("Error processing message: %v", err)
			h.sendError(conn, err.Error())
		}
	}
}

// processMessage processes a single chat message
func (h *ChatHandler) processMessage(ctx context.Context, conn *websocket.Conn, userID, language, content string) error {
	// Classify intent
	result := h.classifier.Classify(content, language)
	log.Printf("Intent: %s (confidence: %.2f)", result.Intent, result.Confidence)

	// Save user message
	if _, err := h.db.SaveMessage(ctx, userID, "user", content); err != nil {
		return fmt.Errorf("failed to save message: %w", err)
	}

	// Add to memory
	h.memoryManager.AddMessage(userID, memory.Message{
		Role:    "user",
		Content: content,
	})

	// Handle small talk without AI
	if result.Intent == classifier.IntentSmallTalk {
		response := h.getSmallTalkResponse(language)
		if err := h.sendMessage(conn, response); err != nil {
			return err
		}
		h.sendDone(conn)
		return nil
	}

	// Check calendar suggestion
	if shouldSuggest := h.calSuggester.ShouldSuggest(result.Intent, content); shouldSuggest.ShouldSuggest {
		suggestion := h.calSuggester.BuildSuggestion(result.Intent, content)
		if err := h.sendCalendarSuggestion(conn, suggestion); err != nil {
			return err
		}
	}

	// Build super-prompt
	facts, _ := h.db.GetUserFacts(ctx, userID)
	shortTermMsgs := h.memoryManager.GetShortTermMemory(userID)

	promptReq := prompt.PromptRequest{
		UserID:          userID,
		UserMessage:     content,
		Language:        language,
		IsSmallTalk:     result.Intent == classifier.IntentSmallTalk,
		ShortTermMemory: shortTermMsgs,
		Facts:           convertDBFactsToMemoryFacts(facts),
	}
	messages := h.promptBuilder.BuildPrompt(promptReq)

	// Call DeepSeek API
	req := deepseek.ChatRequest{
		Model:       "deepseek-chat",
		Messages:    messages,
		Temperature: 0.7,
		MaxTokens:   1000,
		Stream:      true,
	}

	chunks, err := h.deepseekClient.StreamChatCompletion(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to call DeepSeek: %w", err)
	}

	// Stream response
	var fullResponse strings.Builder
	for chunk := range chunks {
		if len(chunk.Choices) > 0 && chunk.Choices[0].Delta.Content != "" {
			content := chunk.Choices[0].Delta.Content
			fullResponse.WriteString(content)
			if err := h.sendMessage(conn, content); err != nil {
				return err
			}
		}
	}

	// Save assistant message
	assistantMsg := fullResponse.String()
	if _, err := h.db.SaveMessage(ctx, userID, "assistant", assistantMsg); err != nil {
		log.Printf("Failed to save assistant message: %v", err)
	}

	// Add to memory
	h.memoryManager.AddMessage(userID, memory.Message{
		Role:    "assistant",
		Content: assistantMsg,
	})

	// Extract facts (simplified - in production, use AI or rules)
	// This is a placeholder for fact extraction logic
	h.extractAndSaveFacts(ctx, userID, content, assistantMsg)

	h.sendDone(conn)
	return nil
}

// convertDBFactsToMemoryFacts converts database facts to memory facts
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

// buildUserContext builds user context for prompt (legacy - not used anymore)
func (h *ChatHandler) buildUserContext(ctx context.Context, userID string) map[string]interface{} {
	user, _ := h.db.GetUserByID(ctx, userID)
	facts, _ := h.db.GetUserFacts(ctx, userID)

	context := make(map[string]interface{})

	if user != nil && user.Name != nil {
		context["name"] = *user.Name
	}

	// Convert facts to map
	factsMap := make(map[string]string)
	for _, fact := range facts {
		factsMap[fact.Key] = fact.Value
	}
	context["facts"] = factsMap

	return context
}

// extractAndSaveFacts extracts facts from conversation (simplified)
func (h *ChatHandler) extractAndSaveFacts(ctx context.Context, userID, userMsg, aiMsg string) {
	// This is a simplified version - in production, use AI to extract facts
	// For now, just look for pregnancy week mentions
	normalized := strings.ToLower(userMsg)

	// Example: detect pregnancy week
	if strings.Contains(normalized, "week") && strings.Contains(normalized, "pregnant") {
		// Simple pattern matching - in production, use proper NLP
		for i := 1; i <= 42; i++ {
			weekStr := fmt.Sprintf("%d week", i)
			if strings.Contains(normalized, weekStr) {
				h.db.SaveOrUpdateFact(ctx, userID, "pregnancy_week", fmt.Sprintf("%d", i), 0.8)
				break
			}
		}
	}
}

// getSmallTalkResponse returns a canned response for small talk
func (h *ChatHandler) getSmallTalkResponse(language string) string {
	responses := map[string]string{
		"en": "I'm here with you. How can I help today?",
		"es": "Estoy aquí contigo. ¿Cómo puedo ayudarte hoy?",
	}

	if resp, ok := responses[language]; ok {
		return resp
	}
	return responses["en"]
}

// sendMessage sends a message chunk to the client
func (h *ChatHandler) sendMessage(conn *websocket.Conn, content string) error {
	return conn.WriteJSON(OutgoingMessage{
		Type:    "message",
		Content: content,
	})
}

// sendCalendarSuggestion sends a calendar suggestion to the client
func (h *ChatHandler) sendCalendarSuggestion(conn *websocket.Conn, suggestion calendar.Suggestion) error {
	return conn.WriteJSON(OutgoingMessage{
		Type: "calendar",
		Data: suggestion,
	})
}

// sendError sends an error message to the client
func (h *ChatHandler) sendError(conn *websocket.Conn, message string) error {
	return conn.WriteJSON(OutgoingMessage{
		Type:    "error",
		Content: message,
	})
}

// sendDone signals that the response is complete
func (h *ChatHandler) sendDone(conn *websocket.Conn) error {
	return conn.WriteJSON(OutgoingMessage{
		Type: "done",
	})
}
