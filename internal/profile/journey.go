package profile

import (
	"fmt"
	"strings"
	"time"
)

const (
	StageTTC         = "ttc"
	StagePregnant    = "pregnant"
	StagePostpartum  = "postpartum"
	StageMiscarriage = "miscarriage"
)

var validStages = map[string]bool{
	StageTTC:         true,
	StagePregnant:    true,
	StagePostpartum:  true,
	StageMiscarriage: true,
}

// AllowedTransitions maps current stage to permitted next stages.
var AllowedTransitions = map[string][]string{
	StageTTC:         {StagePregnant},
	StagePregnant:    {StagePostpartum, StageMiscarriage},
	StageMiscarriage: {StageTTC, StagePregnant},
	StagePostpartum:  {StageTTC, StagePregnant},
}

// StageSaveInput captures journey fields from profile/onboarding requests.
type StageSaveInput struct {
	Stage         string
	PregnancyWeek *int
	ExpectedDue   *time.Time
	BabyBirthDate *time.Time
	LossDate      *time.Time
	IsFirstPreg   *bool
}

// NormalizeStage trims and validates a journey stage value.
func NormalizeStage(stage string) (string, error) {
	stage = strings.TrimSpace(strings.ToLower(stage))
	if stage == "" {
		return "", fmt.Errorf("journey_stage is required")
	}
	if !validStages[stage] {
		return "", fmt.Errorf("invalid journey_stage")
	}
	return stage, nil
}

// CanTransition reports whether moving from one stage to another is allowed.
func CanTransition(from, to string) bool {
	if from == "" || from == to {
		return true
	}
	allowed, ok := AllowedTransitions[from]
	if !ok {
		return false
	}
	for _, next := range allowed {
		if next == to {
			return true
		}
	}
	return false
}

// ValidateStageProfile enforces required fields for onboarding or stage changes.
func ValidateStageProfile(stage string, input StageSaveInput, isOnboarding bool) error {
	if _, err := NormalizeStage(stage); err != nil {
		return err
	}

	switch stage {
	case StagePregnant:
		if input.PregnancyWeek == nil && input.ExpectedDue == nil {
			return fmt.Errorf("pregnancy_week or expected_delivery_date is required for pregnant stage")
		}
	case StagePostpartum:
		if input.BabyBirthDate == nil {
			return fmt.Errorf("baby_birth_date is required for postpartum stage")
		}
	case StageTTC, StageMiscarriage:
		// No additional required fields.
	}

	return nil
}

// WeeksPostpartum returns whole weeks since birth, minimum 0.
func WeeksPostpartum(birthDate time.Time, now time.Time) int {
	birth := dateOnly(birthDate)
	today := dateOnly(now)
	days := int(today.Sub(birth).Hours() / 24)
	if days < 0 {
		return 0
	}
	return days / 7
}

// StageLabel returns a human-readable stage name for prompts.
func StageLabel(stage string) string {
	switch stage {
	case StageTTC:
		return "trying to conceive"
	case StagePregnant:
		return "currently pregnant"
	case StagePostpartum:
		return "postpartum recovery"
	case StageMiscarriage:
		return "pregnancy loss recovery"
	default:
		return stage
	}
}

func dateOnly(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, t.Location())
}
