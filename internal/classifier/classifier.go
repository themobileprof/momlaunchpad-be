package classifier

import (
	"regexp"
	"strings"
)

// Intent represents the classified intent of a user message
type Intent string

const (
	IntentSmallTalk  Intent = "small_talk"
	IntentGratitude  Intent = "gratitude"
	IntentPregnancyQ Intent = "pregnancy_question"
	IntentSymptom    Intent = "symptom_report"
	IntentScheduling Intent = "scheduling_related"
	IntentUnclear    Intent = "unclear"
)

// ClassifierResult contains the classification result
type ClassifierResult struct {
	Intent     Intent  `json:"intent"`
	Confidence float64 `json:"confidence"`
}

// Classifier performs rule-based intent classification
type Classifier struct {
	greetingPatterns   []*regexp.Regexp
	goodbyePatterns    []*regexp.Regexp
	thanksPatterns     []*regexp.Regexp
	pregnancyPatterns  []*regexp.Regexp
	symptomPatterns    []*regexp.Regexp
	schedulingPatterns []*regexp.Regexp
	spaceNormalizer    *regexp.Regexp // Pre-compiled for performance
}

// NewClassifier creates a new intent classifier
func NewClassifier() *Classifier {
	return &Classifier{
		spaceNormalizer: regexp.MustCompile(`\s+`), // Pre-compile once
		greetingPatterns: compilePatterns([]string{
			`\b(hi|hello|hey|hola|buenos días|buenas tardes|good morning|good afternoon)\b`,
			`\bhow are you\b`,
			`\bwhat's up\b`,
			`\bhow's it going\b`,
		}),
		goodbyePatterns: compilePatterns([]string{
			`\b(bye|goodbye|see you|farewell|adiós|hasta luego|chao)\b`,
			`\btalk to you later\b`,
			`\bcatch you later\b`,
		}),
		thanksPatterns: compilePatterns([]string{
			`\b(thanks|thank you|thx|gracias|muchas gracias)\b`,
			`\bappreciate it\b`,
			`\bthanks a lot\b`,
		}),
		pregnancyPatterns: compilePatterns([]string{
			`\b(baby|bebé|fetus|feto|pregnancy|embarazo|pregnant|embarazada)\b`,
			`\b(kick|kicking|movement|moving|moverse|movimiento)\b`,
			`\b(week|weeks|semana|semanas|trimester|trimestre)\b`,
			`\b(develop|development|desarrollo|growth|crecimiento)\b`,
			`\b(ultrasound|ecografía|sonogram)\b`,
			`\b(diet|food|foods|comida|alimentos|eat|eating|comer)\b`,
			`\b(exercise|ejercicio|workout|activity|actividad)\b`,
			`\b(safe|safety|seguro|seguridad)\b`,
			`\bwhat.*(avoid|should|can|is it)\b`,
			`\bwhen will\b`,
			`\bhow often\b`,
			`\bqué.*(debo|puedo|alimentos)\b`,
			`\bcuándo\b`,
		}),
		symptomPatterns: compilePatterns([]string{
			`\b(pain|hurt|hurting|ache|aching|dolor|duele)\b`,
			`\b(nausea|nauseous|náuseas|sick|vomit|vómito)\b`,
			`\b(headache|migraine|dolor de cabeza|migraña)\b`,
			`\b(swelling|swollen|hinchazón|hinchado)\b`,
			`\b(bleeding|blood|spotting|sangrado|sangre)\b`,
			`\b(cramping|cramps|calambres)\b`,
			`\b(dizzy|dizziness|mareo|mareada)\b`,
			`\b(tired|fatigue|exhausted|cansada|fatiga)\b`,
			`\b(fever|fiebre|temperature|temperatura)\b`,
			`\bI('m| am| have).*\b(experiencing|feeling|having|noticing|noticed)\b`,
			`\bmy.*(hurts|aches|is|are)\b`,
			`\btengo\b`,
			`\bme duele\b`,
		}),
		schedulingPatterns: compilePatterns([]string{
			`\b(appointment|cita|visit|visita|checkup|check-up)\b`,
			`\b(remind|reminder|recordatorio|recordar)\b`,
			`\b(schedule|scheduling|programar|agendar)\b`,
			`\b(calendar|calendario)\b`,
			`\b(when is|cuándo es|what time|qué hora)\b`,
			`\bset.*reminder\b`,
			`\bnext appointment\b`,
			`\bpróxima cita\b`,
		}),
	}
}

// Classify determines the intent of the input message
func (c *Classifier) Classify(input, lang string) ClassifierResult {
	normalized := c.normalizeText(input)

	// Empty input handling
	if normalized == "" {
		return ClassifierResult{
			Intent:     IntentUnclear,
			Confidence: 0.1,
		}
	}

	// Check for small talk first (greetings, goodbyes)
	if c.matchesPatterns(normalized, c.greetingPatterns) {
		return ClassifierResult{
			Intent:     IntentSmallTalk,
			Confidence: 0.9,
		}
	}

	if c.matchesPatterns(normalized, c.goodbyePatterns) {
		return ClassifierResult{
			Intent:     IntentSmallTalk,
			Confidence: 0.9,
		}
	}

	// Check for gratitude (handled separately to preserve context)
	if c.matchesPatterns(normalized, c.thanksPatterns) {
		return ClassifierResult{
			Intent:     IntentGratitude,
			Confidence: 0.9,
		}
	}

	// Check for symptom reports (high priority - health concerns)
	symptomMatches := c.countMatches(normalized, c.symptomPatterns)
	if symptomMatches > 0 {
		confidence := 0.75 + float64(symptomMatches)*0.05
		if confidence > 0.95 {
			confidence = 0.95
		}
		return ClassifierResult{
			Intent:     IntentSymptom,
			Confidence: confidence,
		}
	}

	// Check for scheduling
	schedulingMatches := c.countMatches(normalized, c.schedulingPatterns)
	if schedulingMatches > 0 {
		confidence := 0.75 + float64(schedulingMatches)*0.05
		if confidence > 0.95 {
			confidence = 0.95
		}
		return ClassifierResult{
			Intent:     IntentScheduling,
			Confidence: confidence,
		}
	}

	// Check for pregnancy questions
	pregnancyMatches := c.countMatches(normalized, c.pregnancyPatterns)
	if pregnancyMatches > 0 {
		confidence := 0.7 + float64(pregnancyMatches)*0.05
		if confidence > 0.95 {
			confidence = 0.95
		}
		return ClassifierResult{
			Intent:     IntentPregnancyQ,
			Confidence: confidence,
		}
	}

	// Default to unclear if no patterns match
	return ClassifierResult{
		Intent:     IntentUnclear,
		Confidence: 0.3,
	}
}

// normalizeText preprocesses input text for classification
func (c *Classifier) normalizeText(input string) string {
	// Convert to lowercase
	text := strings.ToLower(input)

	// Trim whitespace
	text = strings.TrimSpace(text)

	// Remove multiple spaces using pre-compiled regex
	text = c.spaceNormalizer.ReplaceAllString(text, " ")

	// Remove trailing punctuation
	text = strings.TrimRight(text, "!?.,;:")

	return text
}

// matchesPatterns checks if any pattern matches
func (c *Classifier) matchesPatterns(text string, patterns []*regexp.Regexp) bool {
	for _, pattern := range patterns {
		if pattern.MatchString(text) {
			return true
		}
	}
	return false
}

// countMatches counts how many patterns match
func (c *Classifier) countMatches(text string, patterns []*regexp.Regexp) int {
	count := 0
	for _, pattern := range patterns {
		if pattern.MatchString(text) {
			count++
		}
	}
	return count
}

// compilePatterns compiles a slice of regex patterns
func compilePatterns(patterns []string) []*regexp.Regexp {
	compiled := make([]*regexp.Regexp, 0, len(patterns))
	for _, p := range patterns {
		re := regexp.MustCompile(p)
		compiled = append(compiled, re)
	}
	return compiled
}
