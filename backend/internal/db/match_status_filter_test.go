package db

import (
	"testing"
	"time"
)

func TestMatchStatusWhere(t *testing.T) {
	today := "2026-09-07"
	past := parseDay(t, "2026-09-06")
	future := parseDay(t, "2026-09-08")
	now := parseDay(t, today)

	// Freeze "today" by using same date in tests - matchStatusWhere uses time.Now()
	// Test structural properties instead of exact today branch when time-dependent.

	if got := matchStatusWhere("live", now); got != `m.status = 'live'` {
		t.Fatalf("live: %s", got)
	}
	if got := matchStatusWhere("finished", now); got != `m.status = 'finished'` {
		t.Fatalf("finished: %s", got)
	}
	if got := matchStatusWhere("scheduled", future); got != `m.status = 'scheduled'` {
		t.Fatalf("future scheduled: %s", got)
	}
	if got := matchStatusWhere("", past); got != `m.status = 'finished'` {
		t.Fatalf("past default: %s", got)
	}
	if got := matchStatusWhere("", future); got != `m.status = 'scheduled'` {
		t.Fatalf("future default: %s", got)
	}
}

func TestCouponMatchWhere(t *testing.T) {
	if got := couponMatchWhere(); got != `m.status = 'scheduled' AND m.kickoff_at > NOW()` {
		t.Fatalf("unexpected coupon where: %s", got)
	}
}

func parseDay(t *testing.T, s string) time.Time {
	t.Helper()
	d, err := time.Parse("2006-01-02", s)
	if err != nil {
		t.Fatal(err)
	}
	return d
}
