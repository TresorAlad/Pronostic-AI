package odds

import "testing"

func TestMapToMLKey1X2(t *testing.T) {
	if got := MapToMLKey("Match Winner", "Home"); got != "home_win" {
		t.Fatalf("expected home_win, got %q", got)
	}
	if got := MapToMLKey("Both Teams Score", "Yes"); got != "btts" {
		t.Fatalf("expected btts, got %q", got)
	}
}
