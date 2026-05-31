package profile

import (
	"testing"
	"time"
)

func TestNormalizeStage(t *testing.T) {
	stage, err := NormalizeStage(" Pregnant ")
	if err != nil || stage != StagePregnant {
		t.Fatalf("NormalizeStage = %q, err = %v", stage, err)
	}
	if _, err := NormalizeStage("invalid"); err == nil {
		t.Fatal("expected error for invalid stage")
	}
}

func TestCanTransition(t *testing.T) {
	if !CanTransition(StageTTC, StagePregnant) {
		t.Fatal("ttc -> pregnant should be allowed")
	}
	if CanTransition(StageTTC, StagePostpartum) {
		t.Fatal("ttc -> postpartum should be blocked")
	}
	if !CanTransition(StagePregnant, StageMiscarriage) {
		t.Fatal("pregnant -> miscarriage should be allowed")
	}
	if !CanTransition("", StagePregnant) {
		t.Fatal("initial stage assignment should be allowed")
	}
}

func TestValidateStageProfile(t *testing.T) {
	week := 20
	if err := ValidateStageProfile(StagePregnant, StageSaveInput{PregnancyWeek: &week}, true); err != nil {
		t.Fatalf("pregnant with week: %v", err)
	}
	if err := ValidateStageProfile(StagePregnant, StageSaveInput{}, true); err == nil {
		t.Fatal("expected error without pregnancy timing")
	}
	birth := parseDate("2026-01-01")
	if err := ValidateStageProfile(StagePostpartum, StageSaveInput{BabyBirthDate: &birth}, true); err != nil {
		t.Fatalf("postpartum with birth date: %v", err)
	}
	if err := ValidateStageProfile(StageTTC, StageSaveInput{}, true); err != nil {
		t.Fatalf("ttc should not require extra fields: %v", err)
	}
}

func parseDate(value string) time.Time {
	t, _ := time.Parse("2006-01-02", value)
	return t
}
