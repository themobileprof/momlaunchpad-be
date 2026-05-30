package chat

import (
	"context"
	"fmt"
	"strings"
)

const chatFactConfidence = 0.85

func (e *Engine) extractAndSaveFacts(ctx context.Context, userID, userMsg, aiMsg string) {
	normalized := strings.ToLower(userMsg)

	if week := extractPregnancyWeek(normalized); week > 0 {
		e.db.SaveOrUpdateFact(ctx, userID, "pregnancy_week", fmt.Sprintf("%d", week), chatFactConfidence)
	}

	if value, ok := extractFirstPregnancy(normalized); ok {
		e.db.SaveOrUpdateFact(ctx, userID, "is_first_pregnancy", value, chatFactConfidence)
	}

	if diet := extractDietPreference(normalized); diet != "" {
		e.db.SaveOrUpdateFact(ctx, userID, "diet", diet, chatFactConfidence)
	}

	if concern := extractPrimaryConcernFact(normalized); concern != "" {
		e.db.SaveOrUpdateFact(ctx, userID, "primary_concern", concern, chatFactConfidence)
	}
}

func extractPregnancyWeek(normalized string) int {
	if !strings.Contains(normalized, "week") {
		return 0
	}

	for i := 1; i <= 42; i++ {
		weekStr := fmt.Sprintf("%d week", i)
		if strings.Contains(normalized, weekStr) {
			return i
		}
	}

	return 0
}

func extractFirstPregnancy(normalized string) (string, bool) {
	firstPatterns := []string{
		"first pregnancy",
		"first baby",
		"first time mom",
		"first-time mom",
		"first child",
		"never been pregnant",
	}
	notFirstPatterns := []string{
		"second pregnancy",
		"third pregnancy",
		"not my first",
		"another baby",
		"second baby",
		"third baby",
	}

	for _, pattern := range notFirstPatterns {
		if strings.Contains(normalized, pattern) {
			return "no", true
		}
	}

	for _, pattern := range firstPatterns {
		if strings.Contains(normalized, pattern) {
			return "yes", true
		}
	}

	return "", false
}

func extractDietPreference(normalized string) string {
	dietPatterns := map[string]string{
		"vegetarian": "vegetarian",
		"vegan":      "vegan",
		"pescatarian": "pescatarian",
		"gluten free": "gluten-free",
		"gluten-free": "gluten-free",
		"halal":       "halal",
		"kosher":      "kosher",
	}

	for pattern, value := range dietPatterns {
		if strings.Contains(normalized, pattern) {
			return value
		}
	}

	return ""
}

func extractPrimaryConcernFact(normalized string) string {
	concernKeywords := map[string]string{
		"morning sickness": "morning sickness",
		"nausea":           "nausea",
		"headache":         "headaches",
		"back pain":        "back pain",
		"cramp":            "cramping",
		"heartburn":        "heartburn",
		"swelling":         "swelling",
		"insomnia":         "sleep issues",
		"anxiety":          "anxiety",
		"nutrition":        "nutrition",
		"diet":             "nutrition",
	}

	for keyword, concern := range concernKeywords {
		if strings.Contains(normalized, keyword) {
			return concern
		}
	}

	return ""
}
