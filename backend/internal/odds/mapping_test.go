package odds

import "testing"

func TestMapGoalsOver25(t *testing.T) {
	key := MapToMLKey("Goals Over/Under", "Over 2.5")
	if key != "over_2_5" {
		t.Fatalf("expected over_2_5, got %q", key)
	}
}

func TestMapBTTSNo(t *testing.T) {
	key := MapToMLKey("Both Teams Score", "No")
	if key != "btts_no" {
		t.Fatalf("expected btts_no, got %q", key)
	}
}

func TestMapDoubleChance(t *testing.T) {
	key := MapToMLKey("Double Chance", "Home/Draw")
	if key != "double_chance_1x" {
		t.Fatalf("expected double_chance_1x, got %q", key)
	}
}

func TestMapTeamTotalHome(t *testing.T) {
	key := MapToMLKey("Total - Home", "Over 0.5")
	if key != "team_over_0_5_home" {
		t.Fatalf("expected team_over_0_5_home, got %q", key)
	}
}
