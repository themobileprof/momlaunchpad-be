package api

import (
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/themobileprof/momlaunchpad-be/internal/db"
)

type ConversationHandler struct {
	db *db.DB
}

func NewConversationHandler(database *db.DB) *ConversationHandler {
	return &ConversationHandler{db: database}
}

func (h *ConversationHandler) RegisterRoutes(r *gin.RouterGroup) {
	conversations := r.Group("/conversations")
	conversations.GET("", h.ListConversations)
	conversations.POST("", h.CreateConversation)
	conversations.GET("/:id", h.GetConversation)
	conversations.PATCH("/:id", h.UpdateConversation)
	conversations.DELETE("/:id", h.DeleteConversation)
	conversations.GET("/:id/messages", h.GetMessages)
}

func (h *ConversationHandler) ListConversations(c *gin.Context) {
	userID := c.MustGet("user_id").(string)

	limitStr := c.Query("limit")
	limit := 20
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	offsetStr := c.Query("offset")
	offset := 0
	if offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	conversations, err := h.db.GetConversations(c.Request.Context(), userID, limit, offset)
	if err != nil {
		log.Printf("Failed to get conversations: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve conversations"})
		return
	}

	c.JSON(http.StatusOK, conversations)
}

func (h *ConversationHandler) CreateConversation(c *gin.Context) {
	userID := c.MustGet("user_id").(string)

	var req struct {
		Title string `json:"title"`
	}

	_ = c.ShouldBindJSON(&req) // title is optional; empty body is valid

	title := resolveConversationTitle(req.Title)

	conversation, err := h.db.CreateConversation(c.Request.Context(), userID, title)
	if err != nil {
		log.Printf("Failed to create conversation: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create conversation"})
		return
	}

	c.JSON(http.StatusCreated, conversation)
}

func (h *ConversationHandler) GetConversation(c *gin.Context) {
	userID := c.MustGet("user_id").(string)
	id := c.Param("id")

	conv, err := h.db.GetConversation(c.Request.Context(), id)
	if err != nil {
		log.Printf("Failed to get conversation: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve conversation"})
		return
	}

	if conv == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Conversation not found"})
		return
	}

	if conv.UserID != userID {
		c.JSON(http.StatusNotFound, gin.H{"error": "Conversation not found"}) // Privacy
		return
	}

	c.JSON(http.StatusOK, conv)
}

func (h *ConversationHandler) UpdateConversation(c *gin.Context) {
	userID := c.MustGet("user_id").(string)
	id := c.Param("id")

	// Check ownership
	conv, err := h.db.GetConversation(c.Request.Context(), id)
	if err != nil {
		log.Printf("Failed to check conversation: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}
	if conv == nil || conv.UserID != userID {
		c.JSON(http.StatusNotFound, gin.H{"error": "Conversation not found"})
		return
	}

	var req struct {
		Title     *string `json:"title"`
		IsStarred *bool   `json:"is_starred"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	updatedConv, err := h.db.UpdateConversation(c.Request.Context(), id, req.Title, req.IsStarred)
	if err != nil {
		log.Printf("Failed to update conversation: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update conversation"})
		return
	}

	c.JSON(http.StatusOK, updatedConv)
}

func (h *ConversationHandler) DeleteConversation(c *gin.Context) {
	userID := c.MustGet("user_id").(string)
	id := c.Param("id")

	// Check ownership
	conv, err := h.db.GetConversation(c.Request.Context(), id)
	if err != nil {
		log.Printf("Failed to check conversation: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}
	if conv == nil || conv.UserID != userID {
		c.JSON(http.StatusNotFound, gin.H{"error": "Conversation not found"})
		return
	}

	if err := h.db.DeleteConversation(c.Request.Context(), id); err != nil {
		log.Printf("Failed to delete conversation: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete conversation"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Conversation deleted"})
}

func (h *ConversationHandler) GetMessages(c *gin.Context) {
	userID := c.MustGet("user_id").(string)
	id := c.Param("id")

	// Check ownership
	conv, err := h.db.GetConversation(c.Request.Context(), id)
	if err != nil {
		log.Printf("Failed to check conversation: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}
	if conv == nil || conv.UserID != userID {
		c.JSON(http.StatusNotFound, gin.H{"error": "Conversation not found"})
		return
	}

	limitStr := c.Query("limit")
	limit := 50
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	offsetStr := c.Query("offset")
	offset := 0
	if offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	messages, err := h.db.GetMessagesByConversation(c.Request.Context(), id, limit, offset)
	if err != nil {
		log.Printf("Failed to get messages: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve messages"})
		return
	}

	c.JSON(http.StatusOK, messages)
}

// resolveConversationTitle returns the client title or the default conversation title.
func resolveConversationTitle(title string) *string {
	if title != "" {
		return &title
	}
	defaultTitle := "New Conversation"
	return &defaultTitle
}
