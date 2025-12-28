package privacy

import (
	"regexp"
	"strings"
)

var (
	// Email pattern
	emailRegex = regexp.MustCompile(`\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Z|a-z]{2,}\b`)

	// Phone patterns (US, international, 7-digit local)
	// Matches: 555-123-4567, (555) 123-4567, 555.123.4567, +1-555-123-4567, 555-1234
	phoneRegex = regexp.MustCompile(`(\+\d{1,3}[-.\s]?)?\(?\d{3}\)?[-.\s]?\d{3}[-.\s]\d{4}|\b\d{3}[-.\s]\d{4}\b`)

	// SSN pattern (US)
	ssnRegex = regexp.MustCompile(`\b\d{3}-\d{2}-\d{4}\b`)

	// Credit card pattern (basic) - must have 4 groups
	creditCardRegex = regexp.MustCompile(`\b\d{4}[-\s]\d{4}[-\s]\d{4}[-\s]\d{4}\b`)

	// Medical record number patterns
	medicalIDRegex = regexp.MustCompile(`\b(MRN|Medical Record|Patient ID)[-:\s]*[A-Z0-9]{6,}\b`)
)

// RedactSensitiveData removes PII from text
func RedactSensitiveData(text string) string {
	// Redact emails
	text = emailRegex.ReplaceAllString(text, "[EMAIL]")

	// Redact phone numbers
	text = phoneRegex.ReplaceAllString(text, "[PHONE]")

	// Redact SSN
	text = ssnRegex.ReplaceAllString(text, "[SSN]")

	// Redact credit cards
	text = creditCardRegex.ReplaceAllString(text, "[CARD]")

	// Redact medical IDs
	text = medicalIDRegex.ReplaceAllStringFunc(text, func(s string) string {
		if strings.Contains(strings.ToLower(s), "mrn") ||
			strings.Contains(strings.ToLower(s), "medical") ||
			strings.Contains(strings.ToLower(s), "patient") {
			return "[MEDICAL_ID]"
		}
		return s
	})

	return text
}

// SanitizeForLogging prepares text for safe logging
func SanitizeForLogging(text string) string {
	redacted := RedactSensitiveData(text)

	// Truncate if too long
	if len(redacted) > 200 {
		return redacted[:197] + "..."
	}

	return redacted
}

// SanitizeForAPI removes PII before sending to external APIs
func SanitizeForAPI(text string) string {
	sanitized := RedactSensitiveData(text)

	// Additional sanitization for API calls
	// Remove any remaining numbers that might be sensitive
	// but preserve pregnancy-related numbers (weeks, measurements)

	return sanitized
}

// ContainsPII checks if text contains potential PII
func ContainsPII(text string) bool {
	return emailRegex.MatchString(text) ||
		phoneRegex.MatchString(text) ||
		ssnRegex.MatchString(text) ||
		creditCardRegex.MatchString(text) ||
		medicalIDRegex.MatchString(text)
}

// RedactUserInfo removes user-identifying information
func RedactUserInfo(email, name string) (string, string) {
	// Replace email with hashed version
	redactedEmail := "[USER_" + hashString(email)[:8] + "]"

	// Replace name with placeholder
	redactedName := "[USER]"

	return redactedEmail, redactedName
}

// Simple hash function for generating consistent user IDs
func hashString(s string) string {
	hash := uint32(0)
	for _, c := range s {
		hash = hash*31 + uint32(c)
	}
	return string(rune(hash))
}
