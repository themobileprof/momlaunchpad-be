package calendar

import (
"strings"
"time"

"github.com/themobileprof/momlaunchpad-be/internal/classifier"
)

// SuggestionResult represents the decision on whether to suggest a reminder
type SuggestionResult struct {
ShouldSuggest bool
Priority      string // "urgent", "high", "medium", "low"
}

// Suggestion represents a calendar reminder suggestion
type Suggestion struct {
Type          string    `json:"type"`
Title         string    `json:"title"`
Description   string    `json:"description"`
SuggestedTime time.Time `json:"suggested_time"`
}

// Suggester handles calendar reminder suggestions
type Suggester struct {
urgentKeywords []string
}

// NewSuggester creates a new calendar suggester
func NewSuggester() *Suggester {
return &Suggester{
urgentKeywords: []string{
"severe", "bleeding", "emergency", "urgent",
"intense pain", "can't breathe", "contractions",
},
}
}

// ShouldSuggest determines if a calendar reminder should be suggested
func (s *Suggester) ShouldSuggest(intent classifier.Intent, message string) SuggestionResult {
// Only suggest for symptoms and scheduling
if intent != classifier.IntentSymptom && intent != classifier.IntentScheduling {
return SuggestionResult{
ShouldSuggest: false,
Priority:      "",
}
}

// Check for urgent keywords
lowerMsg := strings.ToLower(message)
for _, keyword := range s.urgentKeywords {
if strings.Contains(lowerMsg, keyword) {
return SuggestionResult{
ShouldSuggest: true,
Priority:      "urgent",
}
}
}

// Default priority for symptoms and scheduling
return SuggestionResult{
ShouldSuggest: true,
Priority:      "high",
}
}

// BuildSuggestion creates a calendar suggestion based on intent and message
func (s *Suggester) BuildSuggestion(intent classifier.Intent, message string) Suggestion {
now := time.Now()

switch intent {
case classifier.IntentSymptom:
return Suggestion{
Type:          "symptom_followup",
Title:         "Follow up on symptom",
Description:   "Check if the symptom persists or improves",
SuggestedTime: now.Add(24 * time.Hour), // Tomorrow
}
case classifier.IntentScheduling:
return Suggestion{
Type:          "appointment",
Title:         "Appointment reminder",
Description:   message,
SuggestedTime: now.Add(1 * time.Hour), // Default to 1 hour from now
}
default:
return Suggestion{}
}
}
