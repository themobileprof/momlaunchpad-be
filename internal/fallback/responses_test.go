package fallback

import (
	"strings"
	"testing"

	"github.com/themobileprof/momlaunchpad-be/internal/classifier"
)

func TestGetFallbackResponse(t *testing.T) {
	tests := []struct {
		name           string
		intent         classifier.Intent
		language       string
		expectedAction string
		containsText   string
	}{
		{
			name:           "English symptom fallback",
			intent:         classifier.IntentSymptom,
			language:       "en",
			expectedAction: "emergency",
			containsText:   "healthcare provider",
		},
		{
			name:           "Spanish symptom fallback",
			intent:         classifier.IntentSymptom,
			language:       "es",
			expectedAction: "emergency",
			containsText:   "proveedor de salud",
		},
		{
			name:           "French symptom fallback",
			intent:         classifier.IntentSymptom,
			language:       "fr",
			expectedAction: "emergency",
			containsText:   "professionnel de santé",
		},
		{
			name:           "English pregnancy question fallback",
			intent:         classifier.IntentPregnancyQ,
			language:       "en",
			expectedAction: "retry",
			containsText:   "connection issue",
		},
		{
			name:           "Spanish pregnancy question fallback",
			intent:         classifier.IntentPregnancyQ,
			language:       "es",
			expectedAction: "retry",
			containsText:   "problema de conexión",
		},
		{
			name:           "French pregnancy question fallback",
			intent:         classifier.IntentPregnancyQ,
			language:       "fr",
			expectedAction: "retry",
			containsText:   "problème de connexion",
		},
		{
			name:           "English small talk fallback",
			intent:         classifier.IntentSmallTalk,
			language:       "en",
			expectedAction: "retry",
			containsText:   "technical hiccup",
		},
		{
			name:           "Spanish small talk fallback",
			intent:         classifier.IntentSmallTalk,
			language:       "es",
			expectedAction: "retry",
			containsText:   "problema técnico",
		},
		{
			name:           "French small talk fallback",
			intent:         classifier.IntentSmallTalk,
			language:       "fr",
			expectedAction: "retry",
			containsText:   "problème technique",
		},
		{
			name:           "Unknown language defaults to English",
			intent:         classifier.IntentPregnancyQ,
			language:       "de",
			expectedAction: "retry",
			containsText:   "connection issue",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response := GetFallbackResponse(tt.intent, tt.language)

			if response.Action != tt.expectedAction {
				t.Errorf("got action %q, want %q", response.Action, tt.expectedAction)
			}

			if !strings.Contains(strings.ToLower(response.Content), strings.ToLower(tt.containsText)) {
				t.Errorf("response %q does not contain %q", response.Content, tt.containsText)
			}
		})
	}
}

func TestGetTimeoutResponse(t *testing.T) {
	tests := []struct {
		name         string
		language     string
		containsText string
	}{
		{
			name:         "English timeout",
			language:     "en",
			containsText: "taking longer",
		},
		{
			name:         "Spanish timeout",
			language:     "es",
			containsText: "tardando más",
		},
		{
			name:         "French timeout",
			language:     "fr",
			containsText: "prends plus de temps",
		},
		{
			name:         "Unknown language defaults to English",
			language:     "de",
			containsText: "taking longer",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response := GetTimeoutResponse(tt.language)

			if response.Action != "retry" {
				t.Errorf("got action %q, want %q", response.Action, "retry")
			}

			if !strings.Contains(strings.ToLower(response.Content), strings.ToLower(tt.containsText)) {
				t.Errorf("response %q does not contain %q", response.Content, tt.containsText)
			}
		})
	}
}

func TestGetCircuitOpenResponse(t *testing.T) {
	tests := []struct {
		name         string
		language     string
		containsText string
	}{
		{
			name:         "English circuit open",
			language:     "en",
			containsText: "temporarily unavailable",
		},
		{
			name:         "Spanish circuit open",
			language:     "es",
			containsText: "temporalmente no disponible",
		},
		{
			name:         "French circuit open",
			language:     "fr",
			containsText: "temporairement indisponible",
		},
		{
			name:         "Unknown language defaults to English",
			language:     "it",
			containsText: "temporarily unavailable",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response := GetCircuitOpenResponse(tt.language)

			if response.Action != "contact_support" {
				t.Errorf("got action %q, want %q", response.Action, "contact_support")
			}

			if !strings.Contains(strings.ToLower(response.Content), strings.ToLower(tt.containsText)) {
				t.Errorf("response %q does not contain %q", response.Content, tt.containsText)
			}
		})
	}
}

func TestIsEmergencyIntent(t *testing.T) {
	tests := []struct {
		name     string
		intent   classifier.Intent
		expected bool
	}{
		{
			name:     "Symptom is emergency",
			intent:   classifier.IntentSymptom,
			expected: true,
		},
		{
			name:     "Pregnancy question is not emergency",
			intent:   classifier.IntentPregnancyQ,
			expected: false,
		},
		{
			name:     "Scheduling is not emergency",
			intent:   classifier.IntentScheduling,
			expected: false,
		},
		{
			name:     "Small talk is not emergency",
			intent:   classifier.IntentSmallTalk,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsEmergencyIntent(tt.intent)
			if result != tt.expected {
				t.Errorf("got %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestAllLanguagesHaveCompleteCoverage(t *testing.T) {
	languages := []string{"en", "es", "fr"}
	intents := []classifier.Intent{
		classifier.IntentSymptom,
		classifier.IntentPregnancyQ,
		classifier.IntentScheduling,
		classifier.IntentSmallTalk,
		classifier.IntentUnclear,
	}

	for _, lang := range languages {
		t.Run("Language_"+lang, func(t *testing.T) {
			for _, intent := range intents {
				response := GetFallbackResponse(intent, lang)

				if response.Content == "" {
					t.Errorf("Missing content for language %s, intent %v", lang, intent)
				}

				if response.Action == "" {
					t.Errorf("Missing action for language %s, intent %v", lang, intent)
				}
			}

			// Check timeout
			timeoutResp := GetTimeoutResponse(lang)
			if timeoutResp.Content == "" {
				t.Errorf("Missing timeout response for language %s", lang)
			}

			// Check circuit open
			circuitResp := GetCircuitOpenResponse(lang)
			if circuitResp.Content == "" {
				t.Errorf("Missing circuit open response for language %s", lang)
			}
		})
	}
}
