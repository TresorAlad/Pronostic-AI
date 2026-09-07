package matches

import (
	"net/http"
	"testing"
	"time"
)

func TestParseMatchDayQuery(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		query      string
		wantDate   string
		wantLeague string
		wantStatus string
		wantErr    bool
	}{
		{
			name:       "defaults",
			query:      "",
			wantDate:   time.Now().UTC().Format("2006-01-02"),
			wantStatus: "",
		},
		{
			name:       "explicit date and filters",
			query:      "date=2026-09-08&league_id=abc&status=scheduled",
			wantDate:   "2026-09-08",
			wantLeague: "abc",
			wantStatus: "scheduled",
		},
		{
			name:    "invalid date",
			query:   "date=08-09-2026",
			wantErr: true,
		},
		{
			name:       "scheduled_only legacy",
			query:      "scheduled_only=true",
			wantDate:   time.Now().UTC().Format("2006-01-02"),
			wantStatus: "scheduled",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			req, _ := http.NewRequest(http.MethodGet, "/matches/today?"+tc.query, nil)
			got, err := parseMatchDayQuery(req)
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.date.Format("2006-01-02") != tc.wantDate {
				t.Fatalf("date = %s, want %s", got.date.Format("2006-01-02"), tc.wantDate)
			}
			if got.leagueID != tc.wantLeague {
				t.Fatalf("leagueID = %q, want %q", got.leagueID, tc.wantLeague)
			}
			if got.status != tc.wantStatus {
				t.Fatalf("status = %q, want %q", got.status, tc.wantStatus)
			}
		})
	}
}
