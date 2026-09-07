package markets

import (
	"strings"
	"testing"
)

func TestGoalLabelsUseFrenchDecimalComma(t *testing.T) {
	for _, key := range GoalKeys() {
		label := Label(key)
		if label == "" {
			t.Fatalf("missing label for %s", key)
		}
		if strings.Contains(label, "Over") || strings.Contains(label, "Under") {
			t.Fatalf("english label for %s: %s", key, label)
		}
		if strings.Contains(key, "over_") && strings.Contains(key, "_5") && !strings.Contains(label, ",") {
			if strings.Contains(label, "1 but") {
				continue
			}
			t.Fatalf("expected comma decimal in %s: %s", key, label)
		}
	}
}

func TestAwayWinLabelAccord(t *testing.T) {
	if Label("away_win") != "Victoire extérieure" {
		t.Fatalf("bad away_win label: %s", Label("away_win"))
	}
}
