package predictions

import "testing"

func TestBuildOddsForAIEmpty(t *testing.T) {
	if rows := buildOddsForAI(nil, nil); rows != nil {
		t.Fatalf("expected nil, got %#v", rows)
	}
}
