package symptoms

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

// Symptom represents a tracked symptom
type Symptom struct {
	ID                 string
	UserID             string
	SymptomType        string
	Description        string
	Severity           string
	Frequency          string
	OnsetTime          string
	AssociatedSymptoms []string
	IsResolved         bool
	ReportedAt         time.Time
	ResolvedAt         *time.Time
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

// ExtractedSymptom contains extracted symptom information
type ExtractedSymptom struct {
	Type               string
	Description        string
	Severity           string
	Frequency          string
	OnsetTime          string
	AssociatedSymptoms []string
}

// Tracker extracts and manages symptom tracking
type Tracker struct{}

// NewTracker creates a new symptom tracker
func NewTracker() *Tracker {
	return &Tracker{}
}

// symptomKeywords maps symptom types to trigger phrases (checked in stable key order via orderedTypes).
var symptomKeywords = map[string][]string{
	"swelling":           {"swollen", "swelling", "puffy", "edema"},
	"nausea":             {"nausea", "nauseous", "morning sickness", "sick", "queasy"},
	"headache":           {"headache", "head ache", "migraine"},
	"back_pain":          {"back pain", "backache", "back ache", "lower back"},
	"cramping":           {"cramp", "cramping", "cramps"},
	"vision_changes":     {"blurry", "blurred vision", "vision", "can't see", "eyesight"},
	"dizziness":          {"dizzy", "lightheaded", "faint"},
	"fatigue":            {"tired", "exhausted", "fatigue", "sleepy"},
	"insomnia":           {"can't sleep", "insomnia", "awake"},
	"heartburn":          {"heartburn", "acid reflux", "indigestion"},
	"vomiting":           {"vomit", "throw up", "throwing up"},
	"constipation":       {"constipated", "constipation"},
	"bleeding":           {"bleed", "bleeding", "spotting", "blood"},
	"contractions":       {"contraction", "contractions", "tightening"},
	"breast_changes":     {"breast", "nipple", "tender"},
	"mood_changes":       {"mood", "emotional", "crying", "anxious", "depressed"},
	"shortness_breath":   {"breath", "breathing", "can't breathe"},
	"frequent_urination": {"pee", "urinate", "bathroom"},
}

// orderedSymptomTypes ensures deterministic detection order.
var orderedSymptomTypes = []string{
	"swelling", "nausea", "headache", "back_pain", "cramping", "vision_changes",
	"dizziness", "fatigue", "insomnia", "heartburn", "vomiting", "constipation",
	"bleeding", "contractions", "breast_changes", "mood_changes", "shortness_breath",
	"frequent_urination",
}

// ExtractSymptoms analyzes a message and extracts symptom information
func (t *Tracker) ExtractSymptoms(message string) []ExtractedSymptom {
	lower := strings.ToLower(message)
	detectedTypes := detectSymptomTypes(lower)

	symptoms := make([]ExtractedSymptom, 0, len(detectedTypes))
	for _, symptomType := range detectedTypes {
		extracted := ExtractedSymptom{
			Type:        symptomType,
			Description: message,
			Severity:    t.extractSeverity(lower),
			Frequency:   t.extractFrequency(lower),
			OnsetTime:   t.extractOnsetTime(lower),
		}

		for _, otherType := range detectedTypes {
			if otherType != symptomType {
				extracted.AssociatedSymptoms = append(extracted.AssociatedSymptoms, otherType)
			}
		}

		symptoms = append(symptoms, extracted)
	}

	return symptoms
}

func detectSymptomTypes(lower string) []string {
	detected := make([]string, 0)
	for _, symptomType := range orderedSymptomTypes {
		for _, keyword := range symptomKeywords[symptomType] {
			if strings.Contains(lower, keyword) {
				detected = append(detected, symptomType)
				break
			}
		}
	}
	return detected
}

func normalizeSymptomText(message string) string {
	return strings.ToLower(strings.TrimSpace(message))
}

// extractSeverity determines severity from message
func (t *Tracker) extractSeverity(message string) string {
	message = normalizeSymptomText(message)

	for _, keyword := range []string{"severe", "really bad", "terrible", "excruciating", "unbearable", "can't handle"} {
		if strings.Contains(message, keyword) {
			return "severe"
		}
	}

	for _, keyword := range []string{"moderate", "uncomfortable", "bothering"} {
		if strings.Contains(message, keyword) {
			return "moderate"
		}
	}

	// "bad" alone — avoid matching inside "really bad" (handled above as severe)
	if strings.Contains(message, " bad") || strings.HasPrefix(message, "bad ") || message == "bad" {
		return "moderate"
	}

	for _, keyword := range []string{"mild", "slight", "little", "bit of"} {
		if strings.Contains(message, keyword) {
			return "mild"
		}
	}

	return "moderate"
}

// extractFrequency determines how often symptom occurs
func (t *Tracker) extractFrequency(message string) string {
	message = normalizeSymptomText(message)

	for _, keyword := range []string{"constant", "all the time", "always", "won't stop", "continuous"} {
		if strings.Contains(message, keyword) {
			return "constant"
		}
	}

	for _, keyword := range []string{"daily", "every day", "everyday"} {
		if strings.Contains(message, keyword) {
			return "daily"
		}
	}

	for _, keyword := range []string{"often", "frequently", "multiple times"} {
		if strings.Contains(message, keyword) {
			return "frequent"
		}
	}

	for _, keyword := range []string{"sometimes", "occasionally", "now and then"} {
		if strings.Contains(message, keyword) {
			return "occasional"
		}
	}

	for _, keyword := range []string{"once", "one time", "just happened"} {
		if strings.Contains(message, keyword) {
			return "once"
		}
	}

	return "occasional"
}

type onsetRule struct {
	re     *regexp.Regexp
	format func([]string) string
}

var onsetRules = []onsetRule{
	{regexp.MustCompile(`(right now|just now|currently|just started(?:\s+now)?|started now)`), firstMatch},
	{regexp.MustCompile(`(today|this morning|this afternoon|this evening)`), firstMatch},
	{regexp.MustCompile(`yesterday`), staticValue("yesterday")},
	{regexp.MustCompile(`(\d+)\s*days?\s*ago`), daysAgo},
	{regexp.MustCompile(`(\d+)\s*weeks?\s*ago`), weeksAgo},
	{regexp.MustCompile(`(?:going on |for )(\d+)\s*weeks?`), weeksAgo},
	{regexp.MustCompile(`this week`), staticValue("this week")},
	{regexp.MustCompile(`last week`), staticValue("last week")},
	{regexp.MustCompile(`(recently|lately)`), firstMatch},
	{regexp.MustCompile(`(few days|couple days|several days)`), firstMatch},
}

func firstMatch(m []string) string {
	if len(m) > 1 && m[1] != "" {
		return m[1]
	}
	if len(m) > 0 {
		return m[0]
	}
	return "unknown"
}

func staticValue(v string) func([]string) string {
	return func([]string) string { return v }
}

func daysAgo(m []string) string {
	if len(m) > 1 {
		return fmt.Sprintf("%s days ago", m[1])
	}
	return firstMatch(m)
}

func weeksAgo(m []string) string {
	if len(m) > 1 {
		return fmt.Sprintf("%s weeks ago", m[1])
	}
	return firstMatch(m)
}

// extractOnsetTime determines when symptom started (first matching rule wins).
func (t *Tracker) extractOnsetTime(message string) string {
	message = normalizeSymptomText(message)

	for _, rule := range onsetRules {
		if match := rule.re.FindStringSubmatch(message); len(match) > 0 {
			return rule.format(match)
		}
	}

	return "unknown"
}

// FormatSymptomForPrompt creates a human-readable symptom description for prompts
func FormatSymptomForPrompt(s Symptom) string {
	parts := []string{s.SymptomType}

	if s.OnsetTime != "" && s.OnsetTime != "unknown" {
		parts = append(parts, "started "+s.OnsetTime)
	}

	if s.Severity != "" {
		parts = append(parts, s.Severity+" severity")
	}

	if s.Frequency != "" && s.Frequency != "occasional" {
		parts = append(parts, s.Frequency)
	}

	if len(s.AssociatedSymptoms) > 0 {
		parts = append(parts, "with "+strings.Join(s.AssociatedSymptoms, ", "))
	}

	return strings.Join(parts, " - ")
}
