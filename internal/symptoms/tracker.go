package symptoms

import (
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

// ExtractSymptoms analyzes a message and extracts symptom information
func (t *Tracker) ExtractSymptoms(message string) []ExtractedSymptom {
	lower := strings.ToLower(message)
	symptoms := make([]ExtractedSymptom, 0)

	// Common pregnancy symptoms with keywords
	symptomPatterns := map[string][]string{
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

	detectedTypes := make([]string, 0)
	for symptomType, keywords := range symptomPatterns {
		for _, keyword := range keywords {
			if strings.Contains(lower, keyword) {
				detectedTypes = append(detectedTypes, symptomType)
				break
			}
		}
	}

	// Extract each detected symptom
	for _, symptomType := range detectedTypes {
		extracted := ExtractedSymptom{
			Type:        symptomType,
			Description: message,
			Severity:    t.extractSeverity(lower),
			Frequency:   t.extractFrequency(lower),
			OnsetTime:   t.extractOnsetTime(lower),
		}

		// Find associated symptoms
		for _, otherType := range detectedTypes {
			if otherType != symptomType {
				extracted.AssociatedSymptoms = append(extracted.AssociatedSymptoms, otherType)
			}
		}

		symptoms = append(symptoms, extracted)
	}

	return symptoms
}

// extractSeverity determines severity from message
func (t *Tracker) extractSeverity(message string) string {
	severeKeywords := []string{"severe", "really bad", "terrible", "excruciating", "unbearable", "can't handle"}
	moderateKeywords := []string{"moderate", "bad", "uncomfortable", "bothering"}
	mildKeywords := []string{"mild", "slight", "little", "bit of"}

	for _, keyword := range severeKeywords {
		if strings.Contains(message, keyword) {
			return "severe"
		}
	}

	for _, keyword := range moderateKeywords {
		if strings.Contains(message, keyword) {
			return "moderate"
		}
	}

	for _, keyword := range mildKeywords {
		if strings.Contains(message, keyword) {
			return "mild"
		}
	}

	return "moderate" // Default
}

// extractFrequency determines how often symptom occurs
func (t *Tracker) extractFrequency(message string) string {
	constantKeywords := []string{"constant", "all the time", "always", "won't stop", "continuous"}
	dailyKeywords := []string{"daily", "every day", "everyday"}
	frequentKeywords := []string{"often", "frequently", "multiple times"}
	occasionalKeywords := []string{"sometimes", "occasionally", "now and then"}
	onceKeywords := []string{"once", "one time", "just happened"}

	for _, keyword := range constantKeywords {
		if strings.Contains(message, keyword) {
			return "constant"
		}
	}

	for _, keyword := range dailyKeywords {
		if strings.Contains(message, keyword) {
			return "daily"
		}
	}

	for _, keyword := range frequentKeywords {
		if strings.Contains(message, keyword) {
			return "frequent"
		}
	}

	for _, keyword := range occasionalKeywords {
		if strings.Contains(message, keyword) {
			return "occasional"
		}
	}

	for _, keyword := range onceKeywords {
		if strings.Contains(message, keyword) {
			return "once"
		}
	}

	return "occasional" // Default
}

// extractOnsetTime determines when symptom started
func (t *Tracker) extractOnsetTime(message string) string {
	// Time patterns with regex
	timePatterns := map[string]*regexp.Regexp{
		"now":       regexp.MustCompile(`(right now|just now|currently)`),
		"today":     regexp.MustCompile(`(today|this morning|this afternoon|this evening)`),
		"yesterday": regexp.MustCompile(`yesterday`),
		"days_ago":  regexp.MustCompile(`(\d+)\s*days?\s*ago`),
		"weeks_ago": regexp.MustCompile(`(\d+)\s*weeks?\s*ago`),
		"this_week": regexp.MustCompile(`this week`),
		"last_week": regexp.MustCompile(`last week`),
		"recently":  regexp.MustCompile(`(recently|lately)`),
		"few_days":  regexp.MustCompile(`(few days|couple days|several days)`),
	}

	for timeframe, pattern := range timePatterns {
		if pattern.MatchString(message) {
			match := pattern.FindStringSubmatch(message)
			if len(match) > 0 {
				return match[0]
			}
			return timeframe
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
