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
	engine          *chat.Engine
	db              *db.DB
	jwtSecret       string
	wsLimiterPerMin int
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
	// Create transport-agnostic engine
	engine := chat.NewEngine(cls, mem, pb, ds, cal, lm, database)

	return &ChatHandler{
		engine:          engine,
		db:              database,
		jwtSecret:       jwtSecret,
		wsLimiterPerMin: 10,
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

		// Create WebSocket responder
		responder := &wsResponder{conn: conn}

		// Delegate to transport-agnostic engine
		req := chat.ProcessRequest{
			UserID:    userID,
			Message:   msg.Content,
			Language:  userLanguage,
			Responder: responder,
		}

		if err := h.engine.ProcessMessage(c.Request.Context(), req); err != nil {
			log.Printf("Error processing message: %v", err)
			h.sendError(conn, "Sorry, I encountered an error processing your message.")
		}
	}
}

// wsResponder implements chat.Responder for WebSocket transport
type wsResponder struct {
	conn *websocket.Conn
}

func (w *wsResponder) SendMessage(content string) error {
	return w.conn.WriteJSON(OutgoingMessage{
		Type:    "message",
		Content: content,
	})
}

func (w *wsResponder) SendCalendarSuggestion(suggestion calendar.Suggestion) error {
	return w.conn.WriteJSON(OutgoingMessage{
		Type: "calendar",
		Data: suggestion,
	})
}

func (w *wsResponder) SendError(message string) error {
	return w.conn.WriteJSON(OutgoingMessage{
		Type:    "error",
		Content: message,
	})
}

func (w *wsResponder) SendDone() error {
	return w.conn.WriteJSON(OutgoingMessage{
		Type: "done",
	})
}

// sendError is a helper for handler-level errors
func (h *ChatHandler) sendError(conn *websocket.Conn, message string) error {
	return conn.WriteJSON(OutgoingMessage{
		Type:    "error",
		Content: message,
	})
}
