package sync

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestTriggerBackendPOSTSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/evaluation/run" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	TriggerBackendPOST(context.Background(), srv.URL, "/api/v1/evaluation/run")
}

func TestTriggerBackendPOSTRelativeURL(t *testing.T) {
	TriggerBackendPOST(context.Background(), "", "/api/v1/evaluation/run")
}
