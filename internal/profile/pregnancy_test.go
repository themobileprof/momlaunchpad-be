package profile

import (
	"testing"
	"time"
)

func TestWeekFromEDD(t *testing.T) {
	now := mustParseDate("2026-01-01")
	edd := mustParseDate("2026-05-15")

	week := WeekFromEDD(edd, now)
	if week < 18 || week > 22 {
		t.Fatalf("expected week around 20, got %d", week)
	}
}

func TestEDDFromWeek(t *testing.T) {
	now := mustParseDate("2026-01-01")
	edd := EDDFromWeek(20, now)
	weekBack := WeekFromEDD(edd, now)
	if weekBack != 20 {
		t.Fatalf("expected round-trip week 20, got %d", weekBack)
	}
}

func mustParseDate(value string) time.Time {
	t, err := time.Parse("2006-01-02", value)
	if err != nil {
		panic(err)
	}
	return t
}
