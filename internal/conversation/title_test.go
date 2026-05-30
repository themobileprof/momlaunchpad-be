package conversation

import "testing"

func TestIsGenericTitle(t *testing.T) {
	tests := []struct {
		title string
		want  bool
	}{
		{"New conversation", true},
		{"New Conversation", true},
		{"Chat May 30, 3:00 PM", true},
		{"Morning sickness at 12 weeks", false},
	}

	for _, tt := range tests {
		if got := IsGenericTitle(tt.title); got != tt.want {
			t.Errorf("IsGenericTitle(%q) = %v, want %v", tt.title, got, tt.want)
		}
	}
}

func TestFallbackTitle(t *testing.T) {
	short := FallbackTitle("Is cramping normal?")
	if short != "Is cramping normal?" {
		t.Fatalf("unexpected short title: %q", short)
	}

	long := FallbackTitle(string(make([]rune, 60)))
	if len([]rune(long)) != 46 { // 45 + ellipsis rune
		t.Fatalf("expected truncated title, got len %d", len([]rune(long)))
	}
}

func TestSanitizeTitle(t *testing.T) {
	got := sanitizeTitle(`  "Swollen ankles tips"  `)
	if got != "Swollen ankles tips" {
		t.Fatalf("sanitizeTitle() = %q", got)
	}
}
