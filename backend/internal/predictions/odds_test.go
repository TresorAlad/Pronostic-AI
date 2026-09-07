package predictions

import (
	"testing"

	"github.com/prono/backend/internal/db"
)

func TestBuildOddsForAIValueEdge(t *testing.T) {
	rows := buildOddsForAI([]db.MatchOdd{
		{Bookmaker: "Bet365", Market: "Match Winner", Selection: "Home", Odd: 2.0},
	}, map[string]float64{"home_win": 0.6})
	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}
	edge, ok := rows[0]["value_edge"].(float64)
	if !ok || edge < 0.09 || edge > 0.11 {
		t.Fatalf("unexpected value_edge: %#v", rows[0]["value_edge"])
	}
}
