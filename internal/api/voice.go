package api

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/themobileprof/momlaunchpad-be/internal/calendar"
	"github.com/themobileprof/momlaunchpad-be/internal/chat"
	"github.com/themobileprof/momlaunchpad-be/internal/db"
	"github.com/themobileprof/momlaunchpad-be/pkg/twilio"
)

// VoiceHandler handles Twilio Voice webhooks
type VoiceHandler struct {
	twilioClient *twilio.VoiceClient
	chatEngine   *chat.Engine
	db           *db.DB
	callSessions *sync.Map // Store call session data (callSid -> session)
}

// VoiceSession stores data for an active voice call
type VoiceSession struct {
	UserID         string
	CallSid        string
	Language       string
	From           string
	Messages       []string // Store conversation history
	ConversationID string
	mu             sync.RWMutex
}

// NewVoiceHandler creates a new voice handler
func NewVoiceHandler(twilioClient *twilio.VoiceClient, chatEngine *chat.Engine, database *db.DB) *VoiceHandler {
	return &VoiceHandler{
		twilioClient: twilioClient,
		chatEngine:   chatEngine,
		db:           database,
		callSessions: &sync.Map{},
	}
}

// HandleIncoming handles incoming voice calls (Twilio webhook)
func (h *VoiceHandler) HandleIncoming(c *gin.Context) {
	// Parse request body
	if err := c.Request.ParseForm(); err != nil {
		log.Printf("Failed to parse form: %v", err)
		c.String(http.StatusBadRequest, "Invalid request")
		return
	}

	// Parse incoming call parameters
	callParams := twilio.ParseIncomingCall(c.Request.Form)

	log.Printf("Incoming call: CallSid=%s, From=%s, To=%s",
		callParams.CallSid, callParams.From, callParams.To)

	// Look up user by phone number
	user, err := h.getUserByPhone(c.Request.Context(), callParams.From)
	if err != nil {
		log.Printf("User not found for phone %s: %v", callParams.From, err)
		// User not registered
		twiml := twilio.NewTwiMLResponse().
			Say("Welcome to MomLaunchpad. Please register through our app to use voice services.", "", "en-US").
			Hangup().
			String()
		c.Header("Content-Type", "application/xml")
		c.String(http.StatusOK, twiml)
		return
	}

	// Create session for this call
	session := &VoiceSession{
		UserID:   user.ID,
		CallSid:  callParams.CallSid,
		Language: user.Language,
		From:     callParams.From,
		Messages: []string{},
	}
	h.callSessions.Store(callParams.CallSid, session)

	// Determine language and voice
	twilioLang := twilio.GetTwilioLanguageCode(user.Language)
	voice := twilio.GetVoiceForLanguage(user.Language)

	// Build webhook URL for gathering speech
	gatherURL := fmt.Sprintf("/api/voice/gather?callSid=%s", callParams.CallSid)

	// Generate greeting TwiML
	greeting := h.getGreeting(user.Language)
	twiml := twilio.NewTwiMLResponse().
		Say(greeting, voice, twilioLang).
		Gather(gatherURL, "speech", twilioLang, 5).
		Say(h.getPrompt(user.Language), voice, twilioLang).
		EndGather().
		Say("I didn't hear anything. Please call back when you're ready.", voice, twilioLang).
		Hangup().
		String()

	c.Header("Content-Type", "application/xml")
	c.String(http.StatusOK, twiml)
}

// HandleGather handles speech input from user (Gather callback)
func (h *VoiceHandler) HandleGather(c *gin.Context) {
	// Parse request body
	if err := c.Request.ParseForm(); err != nil {
		log.Printf("Failed to parse form: %v", err)
		c.String(http.StatusBadRequest, "Invalid request")
		return
	}

	// Parse gather parameters
	gatherParams := twilio.ParseGather(c.Request.Form)
	callSid := c.Query("callSid")

	if callSid == "" {
		callSid = gatherParams.CallSid
	}

	log.Printf("Gather callback: CallSid=%s, Speech=%s", callSid, gatherParams.SpeechResult)

	// Get session
	sessionVal, exists := h.callSessions.Load(callSid)
	if !exists {
		log.Printf("Session not found for CallSid: %s", callSid)
		twiml := twilio.NewTwiMLResponse().
			Say("Session expired. Please call again.", "", "en-US").
			Hangup().
			String()
		c.Header("Content-Type", "application/xml")
		c.String(http.StatusOK, twiml)
		return
	}
	session := sessionVal.(*VoiceSession)

	// Check if user said anything
	speechResult := gatherParams.SpeechResult
	if speechResult == "" {
		// No speech detected, prompt again or hang up
		twilioLang := twilio.GetTwilioLanguageCode(session.Language)
		voice := twilio.GetVoiceForLanguage(session.Language)
		twiml := twilio.NewTwiMLResponse().
			Say("I didn't catch that. Please try again.", voice, twilioLang).
			Redirect(fmt.Sprintf("/api/voice/gather?callSid=%s", callSid)).
			String()
		c.Header("Content-Type", "application/xml")
		c.String(http.StatusOK, twiml)
		return
	}

	// Store message in session
	session.mu.Lock()
	session.Messages = append(session.Messages, speechResult)
	session.mu.Unlock()

	// Process message through chat engine
	responder := NewVoiceResponder(session)
	req := chat.ProcessRequest{
		UserID:         session.UserID,
		ConversationID: session.ConversationID,
		Message:        speechResult,
		Language:       session.Language,
		Responder:      responder,
	}

	if _, err := h.chatEngine.ProcessMessage(c.Request.Context(), req); err != nil {
		log.Printf("Failed to process message: %v", err)
		twilioLang := twilio.GetTwilioLanguageCode(session.Language)
		voice := twilio.GetVoiceForLanguage(session.Language)
		twiml := twilio.NewTwiMLResponse().
			Say("Sorry, I encountered an error. Please try again.", voice, twilioLang).
			Hangup().
			String()
		c.Header("Content-Type", "application/xml")
		c.String(http.StatusOK, twiml)
		return
	}

	// Get AI response from responder
	aiResponse := responder.GetResponse()
	twilioLang := twilio.GetTwilioLanguageCode(session.Language)
	voice := twilio.GetVoiceForLanguage(session.Language)

	// Generate TwiML to speak response and gather next input
	gatherURL := fmt.Sprintf("/api/voice/gather?callSid=%s", callSid)
	twiml := twilio.NewTwiMLResponse().
		Say(aiResponse, voice, twilioLang).
		Gather(gatherURL, "speech", twilioLang, 5).
		Say(h.getContinuePrompt(session.Language), voice, twilioLang).
		EndGather().
		Say(h.getGoodbye(session.Language), voice, twilioLang).
		Hangup().
		String()

	c.Header("Content-Type", "application/xml")
	c.String(http.StatusOK, twiml)
}

// HandleStatus handles call status callbacks (optional)
func (h *VoiceHandler) HandleStatus(c *gin.Context) {
	if err := c.Request.ParseForm(); err != nil {
		log.Printf("Failed to parse form: %v", err)
		c.String(http.StatusBadRequest, "Invalid request")
		return
	}

	callParams := twilio.ParseIncomingCall(c.Request.Form)
	log.Printf("Call status: CallSid=%s, Status=%s", callParams.CallSid, callParams.CallStatus)

	// Clean up session when call ends
	if callParams.CallStatus == twilio.CallStatusCompleted ||
		callParams.CallStatus == twilio.CallStatusFailed ||
		callParams.CallStatus == twilio.CallStatusCanceled {
		h.callSessions.Delete(callParams.CallSid)
	}

	c.String(http.StatusOK, "OK")
}

// VoiceResponder implements chat.Responder for voice calls
type VoiceResponder struct {
	session  *VoiceSession
	response strings.Builder
	mu       sync.Mutex
}

// NewVoiceResponder creates a new voice responder
func NewVoiceResponder(session *VoiceSession) *VoiceResponder {
	return &VoiceResponder{
		session: session,
	}
}

// SetConversationID updates the session with the conversation ID
func (r *VoiceResponder) SetConversationID(id string) {
	r.session.mu.Lock()
	defer r.session.mu.Unlock()
	r.session.ConversationID = id
}

// SendMessage accumulates AI response chunks
func (r *VoiceResponder) SendMessage(content string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.response.WriteString(content)
	return nil
}

// SendCalendarSuggestion handles calendar suggestions in voice
func (r *VoiceResponder) SendCalendarSuggestion(suggestion calendar.Suggestion) error {
	// For voice, we'll just mention it verbally
	message := fmt.Sprintf(" Would you like me to remind you about this? Say yes or no after the beep.")
	r.mu.Lock()
	defer r.mu.Unlock()
	r.response.WriteString(message)
	return nil
}

// SendError sends error message
func (r *VoiceResponder) SendError(message string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.response.WriteString(message)
	return nil
}

// SendDone signals completion
func (r *VoiceResponder) SendDone() error {
	return nil
}

// GetResponse returns accumulated response
func (r *VoiceResponder) GetResponse() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.response.String()
}

// Helper methods for multilingual prompts

func (h *VoiceHandler) getGreeting(language string) string {
	greetings := map[string]string{
		"en": "Welcome to MomLaunchpad, your pregnancy support assistant.",
		"es": "Bienvenida a MomLaunchpad, tu asistente de apoyo durante el embarazo.",
	}
	if greeting, ok := greetings[language]; ok {
		return greeting
	}
	return greetings["en"]
}

func (h *VoiceHandler) getPrompt(language string) string {
	prompts := map[string]string{
		"en": "How can I help you today?",
		"es": "¿Cómo puedo ayudarte hoy?",
	}
	if prompt, ok := prompts[language]; ok {
		return prompt
	}
	return prompts["en"]
}

func (h *VoiceHandler) getContinuePrompt(language string) string {
	prompts := map[string]string{
		"en": "Do you have another question?",
		"es": "¿Tienes otra pregunta?",
	}
	if prompt, ok := prompts[language]; ok {
		return prompt
	}
	return prompts["en"]
}

func (h *VoiceHandler) getGoodbye(language string) string {
	goodbyes := map[string]string{
		"en": "Thank you for calling MomLaunchpad. Take care!",
		"es": "Gracias por llamar a MomLaunchpad. ¡Cuídate!",
	}
	if goodbye, ok := goodbyes[language]; ok {
		return goodbye
	}
	return goodbyes["en"]
}

// getUserByPhone retrieves user by phone number (assumes phone stored in users table)
func (h *VoiceHandler) getUserByPhone(ctx context.Context, phone string) (*db.User, error) {
	// Clean phone number (remove +1, spaces, etc.)
	cleanPhone := strings.ReplaceAll(phone, " ", "")
	cleanPhone = strings.ReplaceAll(cleanPhone, "-", "")
	cleanPhone = strings.ReplaceAll(cleanPhone, "(", "")
	cleanPhone = strings.ReplaceAll(cleanPhone, ")", "")

	// Query user by phone (this assumes a phone_number column exists)
	// For MVP, we can use email or add phone_number column
	query := `
		SELECT id, email, password_hash, display_name, preferred_language, 
		       expected_delivery_date, savings_goal, is_admin, created_at, updated_at
		FROM users
		WHERE email = $1 OR display_name = $2
		LIMIT 1
	`

	user := &db.User{}
	err := h.db.QueryRowContext(ctx, query, phone, phone).Scan(
		&user.ID, &user.Email, &user.PasswordHash, &user.Name,
		&user.Language, &user.ExpectedDeliveryDate, &user.SavingsGoal,
		&user.IsAdmin, &user.CreatedAt, &user.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	return user, nil
}
