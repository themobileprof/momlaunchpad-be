package calendar

import (
	"testing"

	"github.com/themobileprof/momlaunchpad-be/internal/classifier"
)

func TestSuggester_ShouldSuggest(t *testing.T) {
	tests := []struct {
		name         string
		intent       classifier.Intent
		message      string
		wantSuggest  bool
		wantPriority string
	}{
		{
			name:         "symptom report should suggest",
			intent:       classifier.IntentSymptom,
			message:      "I'm experiencing nausea",
			wantSuggest:  true,
			wantPriority: "high",
		},
		{
			name:         "scheduling intent should suggest",
			intent:       classifier.IntentScheduling,
			message:      "I have a doctor appointment next week",
			wantSuggest:  true,
			wantPriority: "high",
		},
		{
			name:         "pregnancy question should not suggest",
			intent:       classifier.IntentPregnancyQ,
			message:      "when will baby kick?",
			wantSuggest:  false,
			wantPriority: "",
		},
		{
			name:         "small talk should not suggest",
			intent:       classifier.IntentSmallTalk,
			message:      "hello",
			wantSuggest:  false,
			wantPriority: "",
		},
		{
			name:         "severe symptom keywords should suggest high priority",
			intent:       classifier.IntentSymptom,
			message:      "severe bleeding and pain",
			wantSuggest:  true,
			wantPriority: "urgent",
		},
	}

	suggester := NewSuggester()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := suggester.ShouldSuggest(tt.intent, tt.message)
			if result.ShouldSuggest != tt.wantSuggest {
				t.Errorf("ShouldSuggest = %v, want %v", result.ShouldSuggest, tt.wantSuggest)
			}
			if result.Priority != tt.wantPriority {
				t.Errorf("Priority = %v, want %v", result.Priority, tt.wantPriority)
			}
		})
	}
}

func TestSuggester_BuildSuggestion(t *testing.T) {
	tests := []struct {
		name     string
		intent   classifier.Intent
		message  string
		wantType string
		hasTitle bool
		hasTime  bool
	}{
		{
			name:     "symptom reminder",
			intent:   classifier.IntentSymptom,
			message:  "I have a headache",
			wantType: "symptom_followup",
			hasTitle: true,
			hasTime:  true,
		},
		{
			name:     "appointment reminder",
			intent:   classifier.IntentScheduling,
			message:  "doctor appointment tomorrow",
			wantType: "appointment",
			hasTitle: true,
			hasTime:  true,
		},
	}

	suggester := NewSuggester()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			suggestion := suggester.BuildSuggestion(tt.intent, tt.message)
			if suggestion.Type != tt.wantType {
				t.Errorf("Type = %v, want %v", suggestion.Type, tt.wantType)
			}
			if tt.hasTitle && suggestion.Title == "" {
				t.Error("expected Title to be non-empty")
			}
			if tt.hasTime && suggestion.SuggestedTime.IsZero() {
				t.Error("expected SuggestedTime to be set")
			}
		})
	}
}
