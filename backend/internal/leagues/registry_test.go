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
	ids := cfg.TrackedExternalIDs()
	if len(ids) != 5 {
		t.Fatalf("expected 5 leagues, got %d", len(ids))
	}
	if !cfg.IsTracked(39) || cfg.IsTracked(999) {
		t.Fatal("IsTracked mismatch")
	}
	if cfg.PrioritySQL("l.external_id") == "" {
		t.Fatal("expected priority SQL")
	}
	couponIDs := cfg.CouponExternalIDs()
	if len(couponIDs) < 5 {
		t.Fatalf("coupon ids too short: %v", couponIDs)
	}
}
