package clients

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type MLClient struct {
	baseURL    string
	httpClient *http.Client
}

func NewMLClient(baseURL string) *MLClient {
	return &MLClient{
		baseURL: baseURL,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

type MLPredictionResponse struct {
	MatchID          string                 `json:"match_id"`
	ModelVersion     string                 `json:"model_version"`
	Predictions      map[string]float64     `json:"predictions"`
	Confidence       map[string]float64     `json:"confidence"`
	NoBetRecommended bool                   `json:"no_bet_recommended"`
	DataSnapshotAt   string                 `json:"data_snapshot_at"`
	IsLive           bool                   `json:"is_live"`
}

func (c *MLClient) PredictMatch(matchID string, features map[string]interface{}, live bool) (*MLPredictionResponse, error) {
	endpoint := "/predict/match"
	if live {
		endpoint = "/predict/match/live"
	}

	body, _ := json.Marshal(map[string]interface{}{
		"match_id": matchID,
		"features": features,
	})

	resp, err := c.httpClient.Post(c.baseURL+endpoint, "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("ml service request: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ml service error %d: %s", resp.StatusCode, string(data))
	}

	var result MLPredictionResponse
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *MLClient) Status() (map[string]interface{}, error) {
	resp, err := c.httpClient.Get(c.baseURL + "/models/status")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	return result, nil
}

type AIAgentClient struct {
	baseURL    string
	httpClient *http.Client
}

func NewAIAgentClient(baseURL string) *AIAgentClient {
	return &AIAgentClient{
		baseURL:    baseURL,
		httpClient: &http.Client{Timeout: 60 * time.Second},
	}
}

type AIAnalysisResponse struct {
	Analysis           string                   `json:"analysis"`
	RecommendedMarkets []map[string]interface{} `json:"recommended_markets"`
	Reasons            []string                 `json:"reasons"`
	Abstain            bool                     `json:"abstain"`
}

func (c *AIAgentClient) AnalyzeMatch(payload map[string]interface{}) (*AIAnalysisResponse, error) {
	body, _ := json.Marshal(payload)
	resp, err := c.httpClient.Post(c.baseURL+"/analyze/match", "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("ai agent request: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ai agent error %d: %s", resp.StatusCode, string(data))
	}

	var result AIAnalysisResponse
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return &result, nil
}
