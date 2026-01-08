package prompt

import (
	"fmt"
	"strings"

	"github.com/themobileprof/momlaunchpad-be/internal/memory"
	"github.com/themobileprof/momlaunchpad-be/pkg/deepseek"
)

// PromptRequest contains all information needed to build a super-prompt
type PromptRequest struct {
	UserID          string
	UserMessage     string
	Language        string
	IsSmallTalk     bool
	ShortTermMemory []memory.Message
	Facts           []memory.UserFact
}

// Builder constructs prompts for the DeepSeek API
type Builder struct {
	// Configuration can be added here if needed
}

// NewBuilder creates a new prompt builder
func NewBuilder() *Builder {
	return &Builder{}
}

// BuildPrompt constructs a super-prompt from the request
func (b *Builder) BuildPrompt(req PromptRequest) []deepseek.ChatMessage {
	// Pre-allocate with estimated capacity (system + history + user)
	capacity := 2 + len(req.ShortTermMemory)
	messages := make([]deepseek.ChatMessage, 0, capacity)

	// For small talk, return minimal prompt
	if req.IsSmallTalk {
		// Small talk can be handled with canned responses
		// But if we need to use AI, keep it minimal
		messages = append(messages, deepseek.ChatMessage{
			Role:    "system",
			Content: "You are a friendly pregnancy support assistant. Keep responses brief and warm.",
		})
		messages = append(messages, deepseek.ChatMessage{
			Role:    "user",
			Content: req.UserMessage,
		})
		return messages
	}

	// Build system prompt with context
	systemPrompt := b.buildSystemPrompt(req)
	messages = append(messages, deepseek.ChatMessage{
		Role:    "system",
		Content: systemPrompt,
	})

	// Add relevant conversation history (skip small talk)
	for _, msg := range req.ShortTermMemory {
		// Only include substantive messages
		if !isLikelySmallTalk(msg.Content) {
			messages = append(messages, deepseek.ChatMessage{
				Role:    msg.Role,
				Content: msg.Content,
			})
		}
	}

	// Add current user message
	messages = append(messages, deepseek.ChatMessage{
		Role:    "user",
		Content: req.UserMessage,
	})

	return messages
}

// buildSystemPrompt creates the system prompt with user context
func (b *Builder) buildSystemPrompt(req PromptRequest) string {
	// Pre-allocate with estimated capacity to reduce allocations
	var sb strings.Builder
	sb.Grow(1024)

	// Base instruction
	sb.WriteString("You are a knowledgeable and empathetic pregnancy support assistant. ")
	sb.WriteString("Your role is to provide accurate, helpful, and supportive information about pregnancy, symptoms, and related topics. ")
	sb.WriteString("\n\n")

	// Voice-friendly response style
	sb.WriteString("RESPONSE STYLE:\n")
	sb.WriteString("- Keep responses brief and conversational (2-4 sentences maximum)\n")
	sb.WriteString("- Speak like a caring friend on a phone call, not a medical textbook\n")
	sb.WriteString("- When symptoms are mentioned, ALWAYS ask 1-2 specific follow-up questions before giving advice\n")
	sb.WriteString("- Ask about: timing, severity, frequency, accompanying symptoms, or what makes it better/worse\n")
	sb.WriteString("- Examples: 'When did this start?', 'How often does this happen?', 'Is there any pain?', 'Does anything make it better?'\n")
	sb.WriteString("\n")

	// Language instruction
	if req.Language == "es" {
		sb.WriteString("Respond in Spanish (Español). ")
	} else if req.Language != "en" {
		sb.WriteString(fmt.Sprintf("Respond in %s if possible, otherwise in English. ", req.Language))
	} else {
		sb.WriteString("Respond in English. ")
	}

	sb.WriteString("\n\n")

	// User context from facts
	if len(req.Facts) > 0 {
		sb.WriteString("User Context:\n")

		// Pregnancy stage (high priority)
		pregnancyWeek := getFactValue(req.Facts, "pregnancy_week")
		if pregnancyWeek != "" {
			sb.WriteString(fmt.Sprintf("- Pregnancy week: %s weeks\n", pregnancyWeek))
		}

		// Other relevant facts
		diet := getFactValue(req.Facts, "diet")
		if diet != "" {
			sb.WriteString(fmt.Sprintf("- Diet: %s\n", diet))
		}

		exercise := getFactValue(req.Facts, "exercise")
		if exercise != "" {
			sb.WriteString(fmt.Sprintf("- Exercise: %s\n", exercise))
		}

		// Include other facts
		for _, fact := range req.Facts {
			if fact.Key != "pregnancy_week" && fact.Key != "diet" && fact.Key != "exercise" {
				sb.WriteString(fmt.Sprintf("- %s: %s\n", fact.Key, fact.Value))
			}
		}

		sb.WriteString("\n")
	}

	// Guidelines
	sb.WriteString("CONVERSATION GUIDELINES:\n")
	sb.WriteString("1. First response to symptom: Ask clarifying questions (timing, severity, etc.)\n")
	sb.WriteString("2. After getting details: Provide brief, reassuring guidance (2-3 sentences max)\n")
	sb.WriteString("3. For concerns: Gently suggest consulting healthcare provider\n")
	sb.WriteString("4. Be warm and supportive, like talking to a close friend\n")
	sb.WriteString("5. Avoid medical jargon - use simple, everyday language\n")

	return sb.String()
}

// getFactValue retrieves a fact value by key
func getFactValue(facts []memory.UserFact, key string) string {
	for _, fact := range facts {
		if fact.Key == key {
			return fact.Value
		}
	}
	return ""
}

// isLikelySmallTalk checks if a message is likely small talk
func isLikelySmallTalk(content string) bool {
	content = strings.ToLower(content)

	smallTalkPatterns := []string{
		"hello", "hi", "hey", "hola",
		"goodbye", "bye", "see you",
		"thanks", "thank you", "gracias",
		"how are you", "what's up",
	}

	// Very short messages are likely small talk
	if len(content) < 15 {
		for _, pattern := range smallTalkPatterns {
			if strings.Contains(content, pattern) {
				return true
			}
		}
	}

	return false
}
