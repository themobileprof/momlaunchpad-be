package conversation

import (
	"context"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/themobileprof/momlaunchpad-be/pkg/llm"
)

const defaultTitle = "New conversation"

// IsGenericTitle reports whether a conversation still has a placeholder title.
func IsGenericTitle(title string) bool {
	t := strings.TrimSpace(title)
	if t == "" {
		return true
	}
	switch strings.ToLower(t) {
	case "new conversation", defaultTitle:
		return true
	}
	return strings.HasPrefix(t, "Chat ")
}

// FallbackTitle builds a short title from the user's first message.
func FallbackTitle(userMessage string) string {
	cleaned := strings.Join(strings.Fields(strings.TrimSpace(userMessage)), " ")
	if cleaned == "" {
		return defaultTitle
	}
	if utf8.RuneCountInString(cleaned) <= 48 {
		return cleaned
	}
	return string([]rune(cleaned)[:45]) + "…"
}

// GenerateTitle asks the LLM for a concise conversation title.
func GenerateTitle(ctx context.Context, client llm.Client, userMessage, assistantMessage string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()

	prompt := fmt.Sprintf(
		"Create a short conversation title (maximum 6 words) for this pregnancy support chat.\n"+
			"User: %s\nAssistant: %s\n\n"+
			"Reply with ONLY the title. No quotes, punctuation at the end, or explanation.",
		truncateForPrompt(userMessage, 300),
		truncateForPrompt(assistantMessage, 300),
	)

	resp, err := client.ChatCompletion(ctx, llm.ChatRequest{
		Messages: []llm.ChatMessage{
			{Role: "user", Content: prompt},
		},
		Temperature: 0.3,
		MaxTokens:   24,
	})
	if err != nil {
		return "", err
	}
	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("empty title response")
	}

	title := sanitizeTitle(resp.Choices[0].Message.Content)
	if title == "" {
		return "", fmt.Errorf("invalid title response")
	}
	return title, nil
}

func sanitizeTitle(raw string) string {
	title := strings.TrimSpace(raw)
	title = strings.Trim(title, "\"'`")
	title = strings.Join(strings.Fields(title), " ")
	if title == "" {
		return ""
	}
	if utf8.RuneCountInString(title) > 60 {
		title = string([]rune(title)[:57]) + "…"
	}
	return title
}

func truncateForPrompt(text string, maxRunes int) string {
	text = strings.TrimSpace(text)
	if utf8.RuneCountInString(text) <= maxRunes {
		return text
	}
	return string([]rune(text)[:maxRunes]) + "…"
}
