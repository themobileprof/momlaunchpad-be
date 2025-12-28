package privacy

import (
	"strings"
	"testing"
)

func TestRedactSensitiveData(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "email redaction",
			input:    "My email is john.doe@example.com",
			expected: "My email is [EMAIL]",
		},
		{
			name:     "phone redaction",
			input:    "Call me at 555-123-4567",
			expected: "Call me at [PHONE]",
		},
		{
			name:     "SSN redaction",
			input:    "My SSN is 123-45-6789",
			expected: "My SSN is [SSN]",
		},
		{
			name:     "credit card redaction",
			input:    "Card: 4532-1234-5678-9010",
			expected: "Card: [CARD]",
		},
		{
			name:     "multiple PII types",
			input:    "Email: test@test.com, Phone: 555-1234",
			expected: "Email: [EMAIL], Phone: [PHONE]",
		},
		{
			name:     "no PII",
			input:    "I'm 14 weeks pregnant",
			expected: "I'm 14 weeks pregnant",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := RedactSensitiveData(tt.input)
			if result != tt.expected {
				t.Errorf("got %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestContainsPII(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "contains email",
			input:    "Contact me at user@example.com",
			expected: true,
		},
		{
			name:     "contains phone",
			input:    "My number is 555-1234",
			expected: true,
		},
		{
			name:     "no PII",
			input:    "I'm feeling nauseous today",
			expected: false,
		},
		{
			name:     "pregnancy info",
			input:    "I'm 14 weeks pregnant with my first baby",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ContainsPII(tt.input)
			if result != tt.expected {
				t.Errorf("got %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestSanitizeForLogging(t *testing.T) {
	longText := strings.Repeat("a", 250)
	result := SanitizeForLogging(longText)

	if len(result) > 200 {
		t.Errorf("result not truncated: got length %d, want <= 200", len(result))
	}

	if !strings.HasSuffix(result, "...") {
		t.Errorf("truncated text should end with '...'")
	}
}
