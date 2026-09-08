package leagues

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadLeaguesConfig(t *testing.T) {
	t.Parallel()

	root, err := filepath.Abs(filepath.Join("..", "..", "..", "config", "leagues.json"))
	if err != nil {
		t.Fatalf("abs path: %v", err)
	}
	if err := os.Setenv("LEAGUES_CONFIG_PATH", root); err != nil {
		t.Fatalf("setenv: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Unsetenv("LEAGUES_CONFIG_PATH")
		ResetForTest()
	})

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.TopMatchesMax() != 10 {
		t.Fatalf("TopMatchesMax = %d", cfg.TopMatchesMax())
	}
	ids := cfg.DisplayExternalIDs()
	if len(ids) != 9 {
		t.Fatalf("expected 9 leagues, got %d", len(ids))
	}
	if ids[0] != 2 {
		t.Fatalf("expected UCL first (id 2), got %d", ids[0])
	}
	if !cfg.IsTracked(39) || cfg.IsTracked(999) {
		t.Fatal("IsTracked mismatch")
	}
	if cfg.PrioritySQL("l.external_id") == "" {
		t.Fatal("expected priority SQL")
	}
	predictable := cfg.PredictableExternalIDs()
	if len(predictable) != 8 {
		t.Fatalf("expected 8 predictable leagues, got %d", len(predictable))
	}
	couponIDs := cfg.CouponExternalIDs()
	if len(couponIDs) != 9 {
		t.Fatalf("coupon ids count: got %d", len(couponIDs))
	}
}
