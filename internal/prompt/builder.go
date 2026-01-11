package prompt

import (
	"fmt"
	"strings"

	"github.com/themobileprof/momlaunchpad-be/internal/conversation"
	"github.com/themobileprof/momlaunchpad-be/internal/memory"
	"github.com/themobileprof/momlaunchpad-be/pkg/deepseek"
)

// PromptRequest contains all information needed to build a super-prompt
type PromptRequest struct {
	UserID            string
	UserMessage       string
	Language          string
	IsSmallTalk       bool
	ShortTermMemory   []memory.Message
	Facts             []memory.UserFact
	RecentSymptoms    []map[string]interface{} // Recent symptom history
	ConversationState *conversation.State      // Track conversation context
	AIName            string                   // AI assistant name (e.g., "MomBot")
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
		aiName := req.AIName
		if aiName == "" {
			aiName = "your pregnancy support assistant"
		}
		messages = append(messages, deepseek.ChatMessage{
			Role:    "system",
			Content: fmt.Sprintf("You are %s. Keep responses brief and warm.", aiName),
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

	// Use custom AI name if provided
	aiName := req.AIName
	if aiName == "" {
		aiName = "a pregnancy support assistant"
	}

	// Base instruction with dynamic name
	sb.WriteString(fmt.Sprintf("You are %s, a knowledgeable and empathetic assistant. ", aiName))
	sb.WriteString("Your role is to provide accurate, helpful, and supportive information about pregnancy, symptoms, and related topics. ")
	sb.WriteString("\n\n")

	// Voice-friendly response style
	sb.WriteString("RESPONSE STYLE:\n")
	sb.WriteString("- Keep responses brief and conversational (2-4 sentences maximum)\n")
	sb.WriteString("- Speak like a caring friend on a phone call, not a medical textbook\n")
	sb.WriteString("- Use simple, everyday language - avoid medical jargon\n")
	sb.WriteString("\n")

	// Conversation flow with state tracking
	sb.WriteString("CONVERSATION FLOW:\n")

	// If there's a primary concern being tracked
	if req.ConversationState != nil && req.ConversationState.PrimaryConcern != "" {
		sb.WriteString(fmt.Sprintf("PRIMARY CONCERN: %s\n", req.ConversationState.PrimaryConcern))
		sb.WriteString("- This is what the user originally asked about - stay focused on resolving this first\n")

		if len(req.ConversationState.SecondaryTopics) > 0 {
			sb.WriteString("- User mentioned side topics, but address them BRIEFLY and return to primary concern\n")
		}

		if req.ConversationState.FollowUpCount >= 2 {
			sb.WriteString("- You've asked enough follow-ups - now provide final advice on PRIMARY CONCERN and conclude\n")
		} else {
			sb.WriteString("- Ask 1-2 clarifying questions about PRIMARY CONCERN only\n")
		}
	} else {
		// Initial message - establish primary concern
		sb.WriteString("- Identify the main concern from user's message\n")
		sb.WriteString("- Ask 1-2 clarifying questions about that ONE topic (timing, severity, etc.)\n")
		sb.WriteString("- Ignore side mentions until primary concern is addressed\n")
	}

	sb.WriteString("\n")
	sb.WriteString("RULES:\n")
	sb.WriteString("1. ONE concern at a time - don't jump between topics\n")
	sb.WriteString("2. After 2 follow-up questions, give advice and conclude\n")
	sb.WriteString("3. If user mentions multiple symptoms, acknowledge but focus on the FIRST/MAIN one\n")
	sb.WriteString("4. Only after resolving primary concern can you address secondary topics\n")
	sb.WriteString("5. End conversations decisively - don't keep asking questions indefinitely\n")
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
			sb.WriteString(fmt.Sprintf("- Pregnancy Week: %s\n", pregnancyWeek))
		}

		// Other relevant facts
		for _, fact := range req.Facts {
			if fact.Key != "pregnancy_week" && fact.Confidence > 0.6 {
				sb.WriteString(fmt.Sprintf("- %s: %s\n", fact.Key, fact.Value))
			}
		}
		sb.WriteString("\n")
	}

	// Recent symptom history (CRITICAL for context and safety)
	if len(req.RecentSymptoms) > 0 {
		sb.WriteString("RECENT SYMPTOM HISTORY (important for tracking patterns):\n")
		for i, symptom := range req.RecentSymptoms {
			if i >= 5 { // Limit to 5 most recent for prompt space
				break
			}

			symptomType := symptom["symptom_type"].(string)
			severity := symptom["severity"].(string)
			frequency := symptom["frequency"].(string)
			onsetTime := symptom["onset_time"].(string)
			isResolved := symptom["is_resolved"].(bool)

			status := "ongoing"
			if isResolved {
				status = "resolved"
			}

			sb.WriteString(fmt.Sprintf("- %s (%s): %s, %s - %s\n",
				symptomType, status, severity, frequency, onsetTime))

			// Include associated symptoms if present
			if assocSymptoms, ok := symptom["associated_symptoms"].([]string); ok && len(assocSymptoms) > 0 {
				sb.WriteString(fmt.Sprintf("  (with: %s)\n", strings.Join(assocSymptoms, ", ")))
			}
		}
		sb.WriteString("\n")
		sb.WriteString("IMPORTANT: Check for patterns or worsening symptoms that may require urgent attention.\n")
		sb.WriteString("If you see RED FLAGS (severe/frequent bleeding, severe headaches + vision changes, severe abdominal pain), advise immediate medical care.\n")
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
