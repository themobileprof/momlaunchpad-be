package community

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/themobileprof/momlaunchpad-be/pkg/llm"
)

// PostAnalysis is AI-derived metadata for a community post.
type PostAnalysis struct {
	Category         string  `json:"category"`
	Scope            string  `json:"scope"`
	MedicalRelevance string  `json:"medical_relevance"`
	IsEvent          bool    `json:"is_event"`
	SafetyFlag       bool    `json:"safety_flag"`
	SpamScore        float32 `json:"spam_score"`
	Status           string  `json:"status"`
}

// Processor classifies posts using LLM with rule-based fallback.
type Processor struct {
	gemini   llm.Client
	deepseek llm.Client
}

// NewProcessor creates a post analysis processor.
func NewProcessor(gemini, deepseek llm.Client) *Processor {
	return &Processor{gemini: gemini, deepseek: deepseek}
}

const analysisPrompt = `Analyze this parenting community post and respond with JSON only (no markdown):
{
  "category": "<one interest key from: first_trimester, second_trimester, third_trimester, pregnancy_health, mental_health, nutrition, fitness, newborn_care, breastfeeding, baby_sleep, baby_health, first_time_moms, experienced_moms, dads_partners, single_parents, ask_midwife, ask_doctor, emotional_support, local_recommendations, local_services, events_meetups, introductions, success_stories>",
  "scope": "local or global",
  "medical_relevance": "none, general, or specialist",
  "is_event": true/false,
  "safety_flag": true if urgent medical/emergency/abuse/self-harm content,
  "spam_score": 0.0 to 1.0,
  "status": "active or pending_review (pending_review if safety_flag or spam_score >= 0.7)"
}

Post:
`

// AnalyzePost returns classification metadata for a post body.
func (p *Processor) AnalyzePost(ctx context.Context, body string) PostAnalysis {
	fallback := ruleBasedAnalysis(body)

	client := p.gemini
	if client == nil {
		client = p.deepseek
	}
	if client == nil {
		return fallback
	}

	resp, err := client.ChatCompletion(ctx, llm.ChatRequest{
		Messages: []llm.ChatMessage{
			{Role: "user", Content: analysisPrompt + body},
		},
		MaxTokens:   256,
		Temperature: 0.1,
	})
	if err != nil || len(resp.Choices) == 0 {
		return fallback
	}

	var analysis PostAnalysis
	content := strings.TrimSpace(resp.Choices[0].Message.Content)
	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimPrefix(content, "```")
	content = strings.TrimSuffix(content, "```")
	if err := json.Unmarshal([]byte(strings.TrimSpace(content)), &analysis); err != nil {
		return fallback
	}

	normalizeAnalysis(&analysis)
	if !IsValidInterest(analysis.Category) {
		analysis.Category = fallback.Category
	}
	return analysis
}

func normalizeAnalysis(a *PostAnalysis) {
	a.Scope = strings.ToLower(strings.TrimSpace(a.Scope))
	if a.Scope != "global" {
		a.Scope = "local"
	}
	a.MedicalRelevance = strings.ToLower(strings.TrimSpace(a.MedicalRelevance))
	switch a.MedicalRelevance {
	case "general", "specialist":
	default:
		a.MedicalRelevance = "none"
	}
	a.Status = strings.ToLower(strings.TrimSpace(a.Status))
	if a.SafetyFlag || a.SpamScore >= 0.7 {
		a.Status = "pending_review"
	} else if a.Status != "pending_review" {
		a.Status = "active"
	}
}

func ruleBasedAnalysis(body string) PostAnalysis {
	lower := strings.ToLower(body)
	analysis := PostAnalysis{
		Category:         "introductions",
		Scope:            "local",
		MedicalRelevance: "none",
		IsEvent:          false,
		SafetyFlag:       false,
		SpamScore:        0,
		Status:           "active",
	}

	eventWords := []string{"workshop", "class", "meetup", "seminar", "event", "gathering"}
	for _, w := range eventWords {
		if strings.Contains(lower, w) {
			analysis.IsEvent = true
			analysis.Category = "events_meetups"
			break
		}
	}

	medicalWords := []string{"doctor", "midwife", "hospital", "bleeding", "pain", "fever"}
	for _, w := range medicalWords {
		if strings.Contains(lower, w) {
			analysis.MedicalRelevance = "general"
			break
		}
	}

	urgentWords := []string{"emergency", "suicide", "kill myself", "abuse", "violence"}
	for _, w := range urgentWords {
		if strings.Contains(lower, w) {
			analysis.SafetyFlag = true
			analysis.Status = "pending_review"
			break
		}
	}

	spamWords := []string{"buy now", "click here", "free money", "crypto"}
	for _, w := range spamWords {
		if strings.Contains(lower, w) {
			analysis.SpamScore = 0.8
			analysis.Status = "pending_review"
			break
		}
	}

	return analysis
}
