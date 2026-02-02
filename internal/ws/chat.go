package ws

import (
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/websocket"
	"github.com/themobileprof/momlaunchpad-be/internal/api/middleware"
	"github.com/themobileprof/momlaunchpad-be/internal/calendar"
	"github.com/themobileprof/momlaunchpad-be/internal/chat"
	"github.com/themobileprof/momlaunchpad-be/internal/db"
	"github.com/themobileprof/momlaunchpad-be/internal/subscription"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for now
	},
}

// ChatHandler handles WebSocket chat connections
type ChatHandler struct {
	engine          *chat.Engine
	db              *db.DB
	jwtSecret       string
	wsLimiterPerMin int
	subManager      *subscription.Manager
}

// NewChatHandler creates a new chat handler
func NewChatHandler(
	engine *chat.Engine,
	database *db.DB,
	jwtSecret string,
	subMgr *subscription.Manager,
) *ChatHandler {
	return &ChatHandler{
		engine:          engine,
		db:              database,
		jwtSecret:       jwtSecret,
		wsLimiterPerMin: 10,
		subManager:      subMgr,
	}
}

// IncomingMessage represents a message from the client
type IncomingMessage struct {
	Content        string `json:"content"`
	ConversationID string `json:"conversation_id,omitempty"`
}

// OutgoingMessage represents a message to the client
type OutgoingMessage struct {
	Type           string      `json:"type"` // "message", "calendar", "error", "done"
	Content        string      `json:"content,omitempty"`
	Data           interface{} `json:"data,omitempty"`
	ConversationID string      `json:"conversation_id,omitempty"`
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

	userLanguage := user.Language

	log.Printf("WebSocket connected: user=%s, language=%s", userID, userLanguage)

	// Create rate limiter for this connection
	wsLimiter := middleware.NewWebSocketLimiter(h.wsLimiterPerMin)

	// Listen for messages
	for {
		var msg IncomingMessage
		if err := conn.ReadJSON(&msg); err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}

		// Rate limiting
		if !wsLimiter.Allow() {
			h.sendError(conn, "Too many messages. Please slow down.")
			continue
		}

		// Check quota before processing
		withinQuota, err := h.subManager.CheckQuota(c.Request.Context(), userID, "chat")
		if err != nil {
			log.Printf("Error checking quota for user %s: %v", userID, err)
			h.sendError(conn, "Sorry, I encountered an error checking your quota.")
			continue
		}
		if !withinQuota {
			h.sendError(conn, "You've reached your message quota for this period. Please upgrade your plan or try again later.")
			continue
		}

		// Create WebSocket responder
		responder := &wsResponder{conn: conn}

		// Delegate to transport-agnostic engine
		req := chat.ProcessRequest{
			UserID:         userID,
			ConversationID: msg.ConversationID,
			Message:        msg.Content,
			Language:       userLanguage,
			Responder:      responder,
		}

		if _, err := h.engine.ProcessMessage(c.Request.Context(), req); err != nil {
			log.Printf("Error processing message: %v", err)
			h.sendError(conn, "Sorry, I encountered an error processing your message.")
			continue
		}

		// Increment usage after successful processing
		if err := h.subManager.IncrementUsage(c.Request.Context(), userID, "chat"); err != nil {
			log.Printf("Error incrementing usage for user %s: %v", userID, err)
			// Don't fail the request, just log the error
		}
	}
}

// wsResponder implements chat.Responder for WebSocket transport
type wsResponder struct {
	conn           *websocket.Conn
	conversationID string
}

func (w *wsResponder) SetConversationID(id string) {
	w.conversationID = id
}

func (w *wsResponder) SendMessage(content string) error {
	return w.conn.WriteJSON(OutgoingMessage{
		Type:           "message",
		Content:        content,
		ConversationID: w.conversationID,
	})
}

func (w *wsResponder) SendCalendarSuggestion(suggestion calendar.Suggestion) error {
	return w.conn.WriteJSON(OutgoingMessage{
		Type:           "calendar",
		Data:           suggestion,
		ConversationID: w.conversationID,
	})
}

func (w *wsResponder) SendError(message string) error {
	return w.conn.WriteJSON(OutgoingMessage{
		Type:           "error",
		Content:        message,
		ConversationID: w.conversationID,
	})
}

func (w *wsResponder) SendDone() error {
	return w.conn.WriteJSON(OutgoingMessage{
		Type:           "done",
		ConversationID: w.conversationID,
	})
}

// sendError is a helper for handler-level errors
func (h *ChatHandler) sendError(conn *websocket.Conn, message string) error {
	return conn.WriteJSON(OutgoingMessage{
		Type:    "error",
		Content: message,
	})
}
