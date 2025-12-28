package classifier

import (
	"testing"
)

func TestClassifier_Classify(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		lang       string
		wantIntent Intent
		minConf    float64
	}{
		// Small Talk - English
		{
			name:       "greeting hello",
			input:      "hello",
			lang:       "en",
			wantIntent: IntentSmallTalk,
			minConf:    0.8,
		},
		{
			name:       "greeting hi",
			input:      "hi there",
			lang:       "en",
			wantIntent: IntentSmallTalk,
			minConf:    0.8,
		},
		{
			name:       "how are you",
			input:      "how are you doing?",
			lang:       "en",
			wantIntent: IntentSmallTalk,
			minConf:    0.8,
		},
		{
			name:       "goodbye",
			input:      "goodbye, see you later",
			lang:       "en",
			wantIntent: IntentSmallTalk,
			minConf:    0.8,
		},
		{
			name:       "thank you",
			input:      "thank you so much",
			lang:       "en",
			wantIntent: IntentSmallTalk,
			minConf:    0.8,
		},

		// Pregnancy Questions - English
		{
			name:       "baby kicking question",
			input:      "when will my baby start kicking?",
			lang:       "en",
			wantIntent: IntentPregnancyQ,
			minConf:    0.7,
		},
		{
			name:       "pregnancy development",
			input:      "what's happening with my baby at 20 weeks?",
			lang:       "en",
			wantIntent: IntentPregnancyQ,
			minConf:    0.7,
		},
		{
			name:       "pregnancy diet",
			input:      "what foods should I avoid during pregnancy?",
			lang:       "en",
			wantIntent: IntentPregnancyQ,
			minConf:    0.7,
		},
		{
			name:       "pregnancy exercise",
			input:      "is it safe to exercise while pregnant?",
			lang:       "en",
			wantIntent: IntentPregnancyQ,
			minConf:    0.7,
		},
		{
			name:       "fetal movement",
			input:      "how often should I feel the baby move?",
			lang:       "en",
			wantIntent: IntentPregnancyQ,
			minConf:    0.7,
		},

		// Symptom Reports - English
		{
			name:       "nausea symptom",
			input:      "I'm experiencing nausea",
			lang:       "en",
			wantIntent: IntentSymptom,
			minConf:    0.7,
		},
		{
			name:       "back pain",
			input:      "my back hurts a lot today",
			lang:       "en",
			wantIntent: IntentSymptom,
			minConf:    0.7,
		},
		{
			name:       "headache report",
			input:      "I have a bad headache",
			lang:       "en",
			wantIntent: IntentSymptom,
			minConf:    0.7,
		},
		{
			name:       "swelling complaint",
			input:      "my feet are swollen",
			lang:       "en",
			wantIntent: IntentSymptom,
			minConf:    0.7,
		},
		{
			name:       "bleeding concern",
			input:      "I noticed some spotting",
			lang:       "en",
			wantIntent: IntentSymptom,
			minConf:    0.7,
		},

		// Scheduling - English
		{
			name:       "appointment question",
			input:      "when is my next appointment?",
			lang:       "en",
			wantIntent: IntentScheduling,
			minConf:    0.7,
		},
		{
			name:       "reminder request",
			input:      "can you remind me to take my vitamins?",
			lang:       "en",
			wantIntent: IntentScheduling,
			minConf:    0.7,
		},
		{
			name:       "schedule ultrasound",
			input:      "I need to schedule an ultrasound",
			lang:       "en",
			wantIntent: IntentScheduling,
			minConf:    0.7,
		},
		{
			name:       "calendar check",
			input:      "what's on my calendar tomorrow?",
			lang:       "en",
			wantIntent: IntentScheduling,
			minConf:    0.7,
		},

		// Small Talk - Spanish
		{
			name:       "spanish hello",
			input:      "hola",
			lang:       "es",
			wantIntent: IntentSmallTalk,
			minConf:    0.8,
		},
		{
			name:       "spanish greeting",
			input:      "buenos días",
			lang:       "es",
			wantIntent: IntentSmallTalk,
			minConf:    0.8,
		},
		{
			name:       "spanish thanks",
			input:      "gracias",
			lang:       "es",
			wantIntent: IntentSmallTalk,
			minConf:    0.8,
		},

		// Pregnancy Questions - Spanish
		{
			name:       "spanish pregnancy question",
			input:      "¿cuándo empezará a moverse mi bebé?",
			lang:       "es",
			wantIntent: IntentPregnancyQ,
			minConf:    0.7,
		},
		{
			name:       "spanish diet question",
			input:      "¿qué alimentos debo evitar durante el embarazo?",
			lang:       "es",
			wantIntent: IntentPregnancyQ,
			minConf:    0.7,
		},

		// Symptoms - Spanish
		{
			name:       "spanish nausea",
			input:      "tengo náuseas",
			lang:       "es",
			wantIntent: IntentSymptom,
			minConf:    0.7,
		},
		{
			name:       "spanish pain",
			input:      "me duele la espalda",
			lang:       "es",
			wantIntent: IntentSymptom,
			minConf:    0.7,
		},

		// Unclear intents
		{
			name:       "ambiguous single word",
			input:      "help",
			lang:       "en",
			wantIntent: IntentUnclear,
			minConf:    0.3,
		},
		{
			name:       "random text",
			input:      "xyz abc 123",
			lang:       "en",
			wantIntent: IntentUnclear,
			minConf:    0.3,
		},
	}

	classifier := NewClassifier()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := classifier.Classify(tt.input, tt.lang)

			if result.Intent != tt.wantIntent {
				t.Errorf("Classify() intent = %v, want %v", result.Intent, tt.wantIntent)
			}

			if result.Confidence < tt.minConf {
				t.Errorf("Classify() confidence = %v, want >= %v", result.Confidence, tt.minConf)
			}
		})
	}
}

func TestClassifier_NormalizeText(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "trim whitespace",
			input: "  hello world  ",
			want:  "hello world",
		},
		{
			name:  "lowercase conversion",
			input: "HELLO World",
			want:  "hello world",
		},
		{
			name:  "remove extra spaces",
			input: "hello    world",
			want:  "hello world",
		},
		{
			name:  "remove punctuation at end",
			input: "hello world!",
			want:  "hello world",
		},
		{
			name:  "preserve internal punctuation",
			input: "I'm feeling good",
			want:  "i'm feeling good",
		},
	}

	classifier := NewClassifier()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := classifier.normalizeText(tt.input)
			if got != tt.want {
				t.Errorf("normalizeText() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestClassifier_EmptyInput(t *testing.T) {
	classifier := NewClassifier()

	result := classifier.Classify("", "en")
	if result.Intent != IntentUnclear {
		t.Errorf("Empty input should return IntentUnclear, got %v", result.Intent)
	}

	if result.Confidence > 0.5 {
		t.Errorf("Empty input confidence should be low, got %v", result.Confidence)
	}
}

func TestClassifier_UnsupportedLanguage(t *testing.T) {
	classifier := NewClassifier()

	// Should still classify based on patterns, even if language is unsupported
	result := classifier.Classify("hello", "fr")
	if result.Intent != IntentSmallTalk {
		t.Errorf("Should detect greeting pattern regardless of language, got %v", result.Intent)
	}
}
