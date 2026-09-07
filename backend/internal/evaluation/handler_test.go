package evaluation

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/prono/backend/internal/db"
)

type fakeEvalStore struct {
	outcomes []db.EvaluationOutcome
}

func (f *fakeEvalStore) EvaluateFinishedPredictions(context.Context) (int, error) {
	return 0, nil
}

func (f *fakeEvalStore) EvaluateUserCoupons(context.Context) (int, error) {
	return 0, nil
}

func (f *fakeEvalStore) RefreshModelPerformance(context.Context) error {
	return nil
}

func (f *fakeEvalStore) GetEvaluationOutcomes(_ context.Context, limit int) ([]db.EvaluationOutcome, error) {
	if limit > len(f.outcomes) {
		return f.outcomes, nil
	}
	return f.outcomes[:limit], nil
}

func (f *fakeEvalStore) NotifyRecentCouponEvaluations(context.Context) error {
	return nil
}

func TestListOutcomesReturnsJSON(t *testing.T) {
	store := &fakeEvalStore{
		outcomes: []db.EvaluationOutcome{
			{
				Market:      "home_win",
				IsCorrect:   true,
				EvaluatedAt: time.Now().Format(time.RFC3339),
			},
		},
	}
	h := &Handler{store: store}

	req := httptest.NewRequest(http.MethodGet, "/evaluation/outcomes?limit=10", nil)
	rec := httptest.NewRecorder()
	h.ListOutcomes(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var items []db.EvaluationOutcome
	if err := json.Unmarshal(rec.Body.Bytes(), &items); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if len(items) != 1 || items[0].Market != "home_win" {
		t.Fatalf("unexpected payload: %#v", items)
	}
}

func TestListOutcomesEmptySlice(t *testing.T) {
	h := &Handler{store: &fakeEvalStore{}}
	req := httptest.NewRequest(http.MethodGet, "/evaluation/outcomes", nil)
	rec := httptest.NewRecorder()
	h.ListOutcomes(rec, req)

	var items []db.EvaluationOutcome
	if err := json.Unmarshal(rec.Body.Bytes(), &items); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if items == nil {
		t.Fatal("expected non-nil empty slice")
	}
}
