package odds

import (
	"testing"
	"time"

	"github.com/prono/backend/internal/db"
)

func TestOddsSummaryForMarket(t *testing.T) {
	rows := []db.MatchOdd{
		{Bookmaker: "Bet365", Market: "Match Winner", Selection: "Home", Odd: 2.1},
		{Bookmaker: "Bet365", Market: "Match Winner", Selection: "Away", Odd: 3.5},
		{Bookmaker: "Unibet", Market: "Match Winner", Selection: "Home", Odd: 2.0},
	}
	summary := OddsSummaryForMarket(rows, "home_win")
	if summary.Best != 2.1 {
		t.Fatalf("expected best 2.1, got %.2f", summary.Best)
	}
	if summary.BookmakerCount != 2 {
		t.Fatalf("expected 2 bookmakers, got %d", summary.BookmakerCount)
	}
	if summary.BestBookmaker != "Bet365" {
		t.Fatalf("expected Bet365, got %s", summary.BestBookmaker)
	}
}

func TestOddTrendForMarket(t *testing.T) {
	now := time.Now()
	history := []db.MatchOddHistory{
		{Market: "Match Winner", Selection: "Home", Odd: 2.0, FetchedAt: now.Add(-2 * time.Hour)},
		{Market: "Match Winner", Selection: "Home", Odd: 2.2, FetchedAt: now.Add(-1 * time.Hour)},
	}
	trend := OddTrendForMarket(history, "home_win")
	if trend.Direction != "up" {
		t.Fatalf("expected up trend, got %s", trend.Direction)
	}
}
