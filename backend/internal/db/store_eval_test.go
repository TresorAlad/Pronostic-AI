package db

import (
	"testing"
)

func TestCouponToExportJSON(t *testing.T) {
	payload := map[string]interface{}{
		"id":   "abc-123",
		"name": "Coupon IA",
		"selections": []map[string]interface{}{
			{"market": "home_win", "confidence": 0.62},
		},
	}
	data, err := CouponToExportJSON(payload)
	if err != nil {
		t.Fatalf("export failed: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("expected non-empty json")
	}
}
