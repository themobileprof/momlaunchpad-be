package symptoms

import (
	"testing"
)

func TestExtractSymptoms(t *testing.T) {
	tracker := NewTracker()

	tests := []struct {
		name           string
		message        string
		expectCount    int
		expectTypes    []string
		expectSeverity string
	}{
		{
			name:           "Single symptom with severity",
			message:        "I've had really bad swollen feet for the past 3 days",
			expectCount:    1,
			expectTypes:    []string{"swelling"},
			expectSeverity: "moderate",
		},
		{
			name:           "Multiple symptoms",
			message:        "I have a headache and nausea, feeling really dizzy",
			expectCount:    3,
			expectTypes:    []string{"headache", "nausea", "dizziness"},
			expectSeverity: "moderate",
		},
		{
			name:        "No symptoms",
			message:     "How are you doing today?",
			expectCount: 0,
		},
		{
			name:           "Severe symptom",
			message:        "I have severe bleeding and unbearable pain",
			expectCount:    2,
			expectTypes:    []string{"bleeding", "back_pain"},
			expectSeverity: "severe",
		},
		{
			name:           "Mild symptom",
			message:        "I have a slight headache, nothing major",
			expectCount:    1,
			expectTypes:    []string{"headache"},
			expectSeverity: "mild",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			symptoms := tracker.ExtractSymptoms(tt.message)

			if len(symptoms) != tt.expectCount {
				t.Errorf("Expected %d symptoms, got %d", tt.expectCount, len(symptoms))
			}

			if len(symptoms) > 0 {
				// Check types
				extractedTypes := make([]string, len(symptoms))
				for i, s := range symptoms {
					extractedTypes[i] = s.Type
				}

				for _, expectedType := range tt.expectTypes {
					found := false
					for _, actualType := range extractedTypes {
						if actualType == expectedType {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("Expected symptom type %s not found in %v", expectedType, extractedTypes)
					}
				}

				// Check severity if specified
				if tt.expectSeverity != "" && symptoms[0].Severity != tt.expectSeverity {
					t.Errorf("Expected severity %s, got %s", tt.expectSeverity, symptoms[0].Severity)
				}
			}
		})
	}
}

func TestExtractSeverity(t *testing.T) {
	tracker := NewTracker()

	tests := []struct {
		message  string
		expected string
	}{
		{"I have severe pain", "severe"},
		{"It's really bad", "severe"},
		{"A moderate headache", "moderate"},
		{"Mild discomfort", "mild"},
		{"Some swelling", "moderate"}, // default
	}

	for _, tt := range tests {
		t.Run(tt.message, func(t *testing.T) {
			result := tracker.extractSeverity(tt.message)
			if result != tt.expected {
				t.Errorf("For '%s': expected %s, got %s", tt.message, tt.expected, result)
			}
		})
	}
}

func TestExtractFrequency(t *testing.T) {
	tracker := NewTracker()

	tests := []struct {
		message  string
		expected string
	}{
		{"It's constant", "constant"},
		{"All the time", "constant"},
		{"Happens daily", "daily"},
		{"Every day", "daily"},
		{"Often occurs", "frequent"},
		{"Sometimes happens", "occasional"},
		{"Happened once", "once"},
		{"Just some pain", "occasional"}, // default
	}

	for _, tt := range tests {
		t.Run(tt.message, func(t *testing.T) {
			result := tracker.extractFrequency(tt.message)
			if result != tt.expected {
				t.Errorf("For '%s': expected %s, got %s", tt.message, tt.expected, result)
			}
		})
	}
}

func TestExtractOnsetTime(t *testing.T) {
	tracker := NewTracker()

	tests := []struct {
		message  string
		expected string
	}{
		{"It started yesterday", "yesterday"},
		{"This happened this morning", "this morning"},
		{"Started 3 days ago", "3 days ago"},
		{"Been going on for 2 weeks", "2 weeks ago"},
		{"Just started now", "right now"},
		{"This week it began", "this week"},
		{"Random text", "unknown"}, // default
	}

	for _, tt := range tests {
		t.Run(tt.message, func(t *testing.T) {
			result := tracker.extractOnsetTime(tt.message)
			if result != tt.expected {
				t.Errorf("For '%s': expected %s, got %s", tt.message, tt.expected, result)
			}
		})
	}
}
