package profile

import "time"

const fullTermWeeks = 40

// WeekFromEDD estimates gestational week from expected delivery date.
func WeekFromEDD(edd time.Time, now time.Time) int {
	daysUntil := int(edd.Sub(now).Hours() / 24)
	weeksRemaining := daysUntil / 7
	week := fullTermWeeks - weeksRemaining
	if week < 1 {
		return 1
	}
	if week > 42 {
		return 42
	}
	return week
}

// EDDFromWeek estimates expected delivery date from current gestational week.
func EDDFromWeek(week int, now time.Time) time.Time {
	if week < 1 {
		week = 1
	}
	if week > 42 {
		week = 42
	}
	weeksRemaining := fullTermWeeks - week
	return now.AddDate(0, 0, weeksRemaining*7)
}

// PregnancyStartFromWeek estimates LMP/start date from gestational week.
func PregnancyStartFromWeek(week int, now time.Time) time.Time {
	if week < 1 {
		week = 1
	}
	if week > 42 {
		week = 42
	}
	return now.AddDate(0, 0, -(week * 7))
}
