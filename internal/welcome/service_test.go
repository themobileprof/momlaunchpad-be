package welcome

import (
	"strings"
	"testing"

	"github.com/themobileprof/momlaunchpad-be/internal/db"
)

func TestFallbackWelcomeUsesFirstName(t *testing.T) {
	name := "Sarah Johnson"
	week := 32
	user := &db.User{
		Name:          &name,
		PregnancyWeek: &week,
	}

	msg := fallbackWelcome(user)
	if !strings.Contains(msg, "Sarah") {
		t.Fatalf("expected name in fallback, got: %s", msg)
	}
	if !strings.Contains(msg, "Week 32") {
		t.Fatalf("expected week in fallback, got: %s", msg)
	}
}

func TestDisplayFirstName(t *testing.T) {
	name := "Sarah"
	user := &db.User{Name: &name}
	if got := displayFirstName(user); got != "Sarah" {
		t.Fatalf("got %q", got)
	}
}
