package sync

import (
	"bytes"
	"context"
	"log"
	"net/http"
	"time"
)

// TriggerBackendPOST appelle un endpoint backend (prewarm, evaluation).
func TriggerBackendPOST(ctx context.Context, baseURL, path string) {
	if baseURL == "" {
		return
	}
	url := baseURL + path
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader([]byte("{}")))
	if err != nil {
		log.Printf("Backend hook request error: %v", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Backend hook %s failed: %v", path, err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		log.Printf("Backend hook %s status %d", path, resp.StatusCode)
		return
	}
	log.Printf("Backend hook OK: %s", path)
}

func TriggerPrewarm(ctx context.Context, baseURL string) {
	TriggerBackendPOST(ctx, baseURL, "/api/v1/predictions/prewarm")
}

func TriggerEvaluation(ctx context.Context, baseURL string) {
	TriggerBackendPOST(ctx, baseURL, "/api/v1/evaluation/run")
}

func TriggerNeo4jSync(ctx context.Context, aiAgentURL string) {
	if aiAgentURL == "" {
		return
	}
	url := aiAgentURL + "/sync/neo4j"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader([]byte("{}")))
	if err != nil {
		log.Printf("Neo4j sync request error: %v", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 300 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Neo4j sync failed: %v", err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		log.Printf("Neo4j sync status %d", resp.StatusCode)
		return
	}
	log.Printf("Neo4j sync OK")
}
