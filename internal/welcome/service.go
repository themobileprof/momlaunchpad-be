package welcome

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/themobileprof/momlaunchpad-be/internal/db"
	"github.com/themobileprof/momlaunchpad-be/pkg/llm"
)

const maxWelcomeWords = 100

// Result is a daily welcome message returned to clients.
type Result struct {
	Message   string
	CacheDate time.Time
	Source    string
}

// Service generates and caches personalized daily welcome messages.
type Service struct {
	db       *db.DB
	gemini   llm.Client
	deepseek llm.Client
}

// NewService creates a welcome message service.
// Either LLM client may be nil; generation falls back to the next provider, then a static template.
func NewService(database *db.DB, gemini, deepseek llm.Client) *Service {
	return &Service{db: database, gemini: gemini, deepseek: deepseek}
}

// GetDailyWelcome returns today's cached message or generates a new one.
func (s *Service) GetDailyWelcome(ctx context.Context, userID string) (*Result, error) {
	cacheDate := startOfDayUTC(time.Now())

	cached, err := s.db.GetWelcomeMessage(ctx, userID, cacheDate)
	if err != nil {
		return nil, err
	}
	if cached != nil {
		return &Result{
			Message:   cached.Message,
			CacheDate: cached.CacheDate,
			Source:    "cache",
		}, nil
	}

	user, err := s.db.GetUserByID(ctx, userID)
	if err != nil || user == nil {
		return nil, fmt.Errorf("user not found")
	}

	contextText, err := s.buildHealthContext(ctx, userID, user)
	if err != nil {
		return nil, err
	}

	message, source, genErr := s.generateMessage(ctx, user, contextText)
	if genErr != nil {
		message = fallbackWelcome(user)
		source = "fallback"
	}

	saved, err := s.db.SaveWelcomeMessage(ctx, userID, cacheDate, message, source)
	if err != nil {
		return &Result{Message: message, CacheDate: cacheDate, Source: source}, nil
	}

	return &Result{
		Message:   saved.Message,
		CacheDate: saved.CacheDate,
		Source:    saved.Source,
	}, nil
}

func (s *Service) generateMessage(ctx context.Context, user *db.User, contextText string) (string, string, error) {
	prompt := buildWelcomePrompt(user, contextText)

	if s.gemini != nil {
		message, err := s.completeWithLLM(ctx, s.gemini, prompt)
		if err == nil {
			return message, "gemini", nil
		}
	}

	if s.deepseek != nil {
		message, err := s.completeWithLLM(ctx, s.deepseek, prompt)
		if err == nil {
			return message, "deepseek", nil
		}
	}

	if s.gemini == nil && s.deepseek == nil {
		return fallbackWelcome(user), "fallback", fmt.Errorf("no LLM client configured")
	}

	return fallbackWelcome(user), "fallback", fmt.Errorf("all LLM providers failed")
}

func buildWelcomePrompt(user *db.User, contextText string) string {
	name := displayFirstName(user)
	weekLine := ""
	if user.PregnancyWeek != nil {
		weekLine = fmt.Sprintf("They are at pregnancy week %d.\n", *user.PregnancyWeek)
	}

	return fmt.Sprintf(`You are a warm, supportive pregnancy companion in a mobile app.

Write ONE personalized welcome message for the user below.

Rules:
- Greet them by first name (%s)
- %sKeep it warm, encouraging, and specific to their context when data is available
- Reference at most 2 health items from the context (symptoms, vitals, visits, appointments) — supportive tone, never alarming
- Do NOT invent medical facts not present in the context
- Maximum %d words (~4 short sentences). Must fit on a phone screen without scrolling
- At most one emoji
- Output ONLY the message text, no quotes or labels

User health context:
%s`, name, weekLine, maxWelcomeWords, contextText)
}

func (s *Service) completeWithLLM(ctx context.Context, client llm.Client, prompt string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()

	resp, err := client.ChatCompletion(ctx, llm.ChatRequest{
		Messages: []llm.ChatMessage{
			{Role: "user", Content: prompt},
		},
		Temperature: 0.75,
		MaxTokens:   220,
	})
	if err != nil {
		return "", err
	}
	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("empty LLM response")
	}

	message := strings.TrimSpace(resp.Choices[0].Message.Content)
	message = strings.Trim(message, "\"'`")
	if message == "" {
		return "", fmt.Errorf("empty welcome text")
	}

	return message, nil
}

func (s *Service) buildHealthContext(ctx context.Context, userID string, user *db.User) (string, error) {
	var b strings.Builder

	b.WriteString("Profile:\n")
	b.WriteString(fmt.Sprintf("- Name: %s\n", displayName(user)))
	if user.PregnancyWeek != nil {
		b.WriteString(fmt.Sprintf("- Pregnancy week: %d\n", *user.PregnancyWeek))
	}
	if user.ExpectedDeliveryDate != nil {
		b.WriteString(fmt.Sprintf("- Expected delivery: %s\n", user.ExpectedDeliveryDate.Format("2006-01-02")))
	}
	if user.PrimaryConcern != nil && *user.PrimaryConcern != "" {
		b.WriteString(fmt.Sprintf("- Primary concern: %s\n", *user.PrimaryConcern))
	}
	if user.IsFirstPregnancy != nil {
		if *user.IsFirstPregnancy {
			b.WriteString("- First pregnancy: yes\n")
		} else {
			b.WriteString("- First pregnancy: no\n")
		}
	}

	facts, _ := s.db.GetUserFacts(ctx, userID)
	if len(facts) > 0 {
		b.WriteString("\nKnown facts:\n")
		for i, fact := range facts {
			if i >= 8 {
				break
			}
			b.WriteString(fmt.Sprintf("- %s: %s\n", fact.Key, fact.Value))
		}
	}

	symptoms, _ := s.db.GetRecentSymptoms(ctx, userID, 8)
	if len(symptoms) > 0 {
		b.WriteString("\nRecent symptoms:\n")
		for i, s := range symptoms {
			if i >= 6 {
				break
			}
			b.WriteString(fmt.Sprintf("- %v (%v, resolved=%v)\n",
				s["symptom_type"], s["severity"], s["is_resolved"]))
		}
	}

	vitals, _ := s.db.GetUserVitalReadings(ctx, userID, 5)
	if len(vitals) > 0 {
		b.WriteString("\nRecent vital readings:\n")
		for i, v := range vitals {
			if i >= 4 {
				break
			}
			line := fmt.Sprintf("- %s:", v.RecordedAt.Format("2006-01-02"))
			if v.BloodPressureSystolic != nil && v.BloodPressureDiastolic != nil {
				line += fmt.Sprintf(" BP %d/%d", *v.BloodPressureSystolic, *v.BloodPressureDiastolic)
			}
			if v.WeightKg != nil {
				line += fmt.Sprintf(" weight %.1f kg", *v.WeightKg)
			}
			if v.HeartRateBpm != nil {
				line += fmt.Sprintf(" HR %d", *v.HeartRateBpm)
			}
			b.WriteString(line + "\n")
		}
	}

	visits, _ := s.db.GetUserDoctorVisits(ctx, userID)
	if len(visits) > 0 {
		b.WriteString("\nRecent doctor visits:\n")
		for i, visit := range visits {
			if i >= 3 {
				break
			}
			line := fmt.Sprintf("- %s (%s)", visit.VisitDate.Format("2006-01-02"), visit.VisitType)
			if visit.Diagnosis != nil && *visit.Diagnosis != "" {
				line += fmt.Sprintf(": %s", truncate(*visit.Diagnosis, 80))
			}
			b.WriteString(line + "\n")
			if visit.NextAppointmentAt != nil {
				b.WriteString(fmt.Sprintf("  Next appointment: %s\n", visit.NextAppointmentAt.Format("2006-01-02")))
			}
		}
	}

	reminders, _ := s.db.GetUserReminders(ctx, userID)
	now := time.Now()
	upcoming := 0
	for _, r := range reminders {
		if !r.IsCompleted && r.ReminderTime.After(now) && upcoming < 3 {
			b.WriteString(fmt.Sprintf("\nUpcoming reminder: %s on %s\n",
				r.Title, r.ReminderTime.Format("2006-01-02")))
			upcoming++
		}
	}

	if b.Len() == 0 {
		return "No detailed health records yet.", nil
	}

	return b.String(), nil
}

func fallbackWelcome(user *db.User) string {
	name := displayFirstName(user)
	week := ""
	if user.PregnancyWeek != nil {
		week = fmt.Sprintf(" Week %d is a big milestone—", *user.PregnancyWeek)
	}
	return fmt.Sprintf(
		"Hi %s!%syou and your baby are doing great work together. Keep up your prenatal care, take medications as prescribed, and reach out to your care team with any concerns.",
		name, week,
	)
}

func displayName(user *db.User) string {
	if user.Name != nil && strings.TrimSpace(*user.Name) != "" {
		return strings.TrimSpace(*user.Name)
	}
	return "User"
}

func displayFirstName(user *db.User) string {
	name := displayName(user)
	parts := strings.Fields(name)
	if len(parts) == 0 {
		return "there"
	}
	return parts[0]
}

func truncate(text string, max int) string {
	text = strings.TrimSpace(text)
	if len(text) <= max {
		return text
	}
	return text[:max] + "…"
}

func startOfDayUTC(t time.Time) time.Time {
	y, m, d := t.UTC().Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}
