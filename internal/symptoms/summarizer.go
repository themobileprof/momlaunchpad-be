package symptoms

import (
	"context"
	"fmt"
	"strings"
	"time"
	"unicode"

	"github.com/themobileprof/momlaunchpad-be/pkg/llm"
)

const maxSummaryWords = 25

// Summarizer generates one-sentence symptom summaries for health tracker UI.
type Summarizer struct {
	gemini   llm.Client
	deepseek llm.Client
}

// NewSummarizer creates a summarizer. Either LLM client may be nil.
func NewSummarizer(gemini, deepseek llm.Client) *Summarizer {
	return &Summarizer{gemini: gemini, deepseek: deepseek}
}

// Summarize returns a one-sentence summary, falling back to a template if LLM fails.
func (s *Summarizer) Summarize(
	ctx context.Context,
	symptomType, description, severity, frequency string,
) string {
	if summary, err := s.summarizeWithLLM(ctx, symptomType, description, severity, frequency); err == nil {
		return summary
	}
	return FallbackSummary(symptomType, description, severity)
}

// TrySummarize attempts an LLM summary; returns fallback text and false on failure.
func (s *Summarizer) TrySummarize(
	ctx context.Context,
	symptomType, description, severity, frequency string,
) (string, bool) {
	summary, err := s.summarizeWithLLM(ctx, symptomType, description, severity, frequency)
	if err != nil {
		return FallbackSummary(symptomType, description, severity), false
	}
	return summary, true
}

func (s *Summarizer) summarizeWithLLM(
	ctx context.Context,
	symptomType, description, severity, frequency string,
) (string, error) {
	prompt := buildSummaryPrompt(symptomType, description, severity, frequency)

	if s.gemini != nil {
		if summary, err := s.complete(ctx, s.gemini, prompt); err == nil {
			return summary, nil
		}
	}
	if s.deepseek != nil {
		if summary, err := s.complete(ctx, s.deepseek, prompt); err == nil {
			return summary, nil
		}
	}
	return "", fmt.Errorf("no LLM available")
}

func buildSummaryPrompt(symptomType, description, severity, frequency string) string {
	typeLabel := strings.ReplaceAll(symptomType, "_", " ")
	return fmt.Sprintf(`Summarize this pregnancy symptom in ONE clear sentence for a health tracker (maximum %d words).

Focus on what the patient is experiencing — not advice, questions, or chat context.
Use plain language a patient would understand.

Type: %s
Severity: %s
Frequency: %s
Patient message: %q

Output ONLY the summary sentence, no quotes or labels.`,
		maxSummaryWords, typeLabel, severity, frequency, description)
}

func (s *Summarizer) complete(ctx context.Context, client llm.Client, prompt string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 12*time.Second)
	defer cancel()

	resp, err := client.ChatCompletion(ctx, llm.ChatRequest{
		Messages:    []llm.ChatMessage{{Role: "user", Content: prompt}},
		Temperature: 0.4,
		MaxTokens:   80,
	})
	if err != nil {
		return "", err
	}
	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("empty LLM response")
	}

	summary := strings.TrimSpace(resp.Choices[0].Message.Content)
	summary = strings.Trim(summary, "\"'`")
	summary = strings.Join(strings.Fields(summary), " ")
	if summary == "" {
		return "", fmt.Errorf("empty summary")
	}
	return summary, nil
}

// FallbackSummary builds a readable one-liner without LLM.
func FallbackSummary(symptomType, description, severity string) string {
	typeLabel := titleWords(strings.ReplaceAll(symptomType, "_", " "))
	desc := strings.TrimSpace(description)
	if desc == "" {
		return fmt.Sprintf("%s (%s severity)", typeLabel, severity)
	}
	if len(desc) > 100 {
		desc = desc[:97] + "..."
	}
	return fmt.Sprintf("%s (%s): %s", typeLabel, severity, desc)
}

func titleWords(s string) string {
	words := strings.Fields(strings.ToLower(s))
	for i, word := range words {
		runes := []rune(word)
		if len(runes) == 0 {
			continue
		}
		runes[0] = unicode.ToUpper(runes[0])
		words[i] = string(runes)
	}
	return strings.Join(words, " ")
}
